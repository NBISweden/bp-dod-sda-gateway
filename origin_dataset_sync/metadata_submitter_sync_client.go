package origin_dataset_sync

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/NBISweden/bp-dod-sda-gateway/models"
	"github.com/NBISweden/bp-dod-sda-gateway/models/metadata_models"
	"github.com/NBISweden/bp-dod-sda-gateway/pkg/observability"
	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/lestrrat-go/jwx/v4/jwk"
	"github.com/lestrrat-go/jwx/v4/jwt"
	"go.opentelemetry.io/otel/attribute"
)

type originDataset struct {
	models.OriginDataset
	files []*fileInfo
}

type fileInfo struct {
	Accession        string
	Path             string
	DatasetAccession string
}

func (ods *originDatasetSync) getOriginDatasetMetadata(ctx context.Context, datasetAccession string) (*originDataset, error) {
	ctx, span := observability.StartSpan(ctx, "getOriginDatasetMetadata", attribute.String("dataset-accession", datasetAccession))
	defer span.End()

	syncUrl, err := url.JoinPath(metadataSubmitterURL, "sync", datasetAccession)
	if err != nil {
		return nil, fmt.Errorf("failed to build sync API path: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, syncUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build http request: %w", err)
	}
	token, err := ods.generateAndSignToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate and sign token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rsp, err := ods.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do http request: %w", err)
	}
	defer func() {
		_ = rsp.Body.Close()
	}()

	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get dataset metadata unexpected status code: %s", rsp.Status)
	}

	zipData, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create zip reader from response body: %w", err)
	}

	od := new(originDataset)
	od.Accession = datasetAccession

	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s from zip reader: %w", f.Name, err)
		}

		switch f.Name {
		case "METADATA/dataset.xml":
			xmlDecoder := xml.NewDecoder(rc)
			err = xmlDecoder.Decode(&od.Dataset)
		case "METADATA/image.xml":
			xmlDecoder := xml.NewDecoder(rc)
			err = xmlDecoder.Decode(&od.Image)
		case "METADATA/annotation.xml":
			xmlDecoder := xml.NewDecoder(rc)
			err = xmlDecoder.Decode(&od.Annotation)
		case "METADATA/observation.xml":
			xmlDecoder := xml.NewDecoder(rc)
			err = xmlDecoder.Decode(&od.Observation)
		case "METADATA/observer.xml":
			xmlDecoder := xml.NewDecoder(rc)
			err = xmlDecoder.Decode(&od.Observer)
		case "METADATA/policy.xml":
			xmlDecoder := xml.NewDecoder(rc)
			err = xmlDecoder.Decode(&od.Policy)
		case "METADATA/sample.xml":
			xmlDecoder := xml.NewDecoder(rc)
			err = xmlDecoder.Decode(&od.Sample)
		case "METADATA/staining.xml":
			xmlDecoder := xml.NewDecoder(rc)
			err = xmlDecoder.Decode(&od.Staining)
		case "METADATA/rems.xml":
			xmlDecoder := xml.NewDecoder(rc)
			var remsEntry metadata_models.RemsSet
			if err = xmlDecoder.Decode(&remsEntry); err != nil {
				break
			}
			if len(remsEntry.Rems) != 1 {
				err = errors.New("rems set has more than one rems entry")
			}
			workflowID, parseErr := strconv.Atoi(remsEntry.Rems[0].WorkflowId)
			if parseErr != nil {
				err = fmt.Errorf("failed to parse rems workflow id to an integer: %w", parseErr)

				break
			}
			od.RemsWorkflowID = workflowID
			od.RemsOrganisationID = remsEntry.Rems[0].OrganisationId
		case "FILES/files.json":
			type file struct {
				FileID string `json:"fileId"`
				Path   string `json:"path"`
			}
			var files []*file

			jsonDecoder := json.NewDecoder(rc)
			if err = jsonDecoder.Decode(&files); err != nil {
				break
			}
			for _, f := range files {
				od.files = append(od.files, &fileInfo{
					Accession:        f.FileID,
					Path:             f.Path,
					DatasetAccession: datasetAccession,
				})
			}

		default:
		}

		_ = rc.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal file %s: %w", f.Name, err)
		}
	}

	return od, nil
}

func (ods *originDatasetSync) generateAndSignToken() (string, error) {
	claims := map[string]any{
		jwt.ExpirationKey: time.Now().UTC().Add(15 * time.Second),
		jwt.IssuedAtKey:   time.Now().UTC(),
		jwt.IssuerKey:     "bp-dod-sda-gateway",
		jwt.SubjectKey:    "bp-dod-sda-gateway",
		jwt.AudienceKey:   "bp-sd-submit-api",
	}

	jwtKey, err := jwk.ParseKey(privateKey, jwk.WithX509(true))
	if err != nil {
		return "", err
	}
	if err := jwtKey.Set(jwk.AlgorithmKey, "ES256"); err != nil {
		return "", err
	}
	if err := jwk.AssignKeyID(jwtKey); err != nil {
		return "", err
	}

	token := jwt.New()
	for key, value := range claims {
		if err := token.Set(key, value); err != nil {
			return "", err
		}
	}

	tokenString, err := jwt.Sign(token, jwt.WithKey(jwa.ES256(), jwtKey))
	if err != nil {
		return "", err
	}

	return string(tokenString), nil
}
