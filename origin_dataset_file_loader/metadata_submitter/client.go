package metadata_submitter

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/NBISweden/bp-dod-sda-gateway/models"
	"github.com/NBISweden/bp-dod-sda-gateway/models/metadata_models"
	"github.com/NBISweden/bp-dod-sda-gateway/origin_dataset_file_loader"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type metadataSubmitterClient struct {
	sync.RWMutex

	client *http.Client

	ctx    context.Context
	cancel context.CancelFunc
	cache  map[string]*cacheEntry
}

type cacheEntry struct {
	lastAccessed  time.Time
	originDataset *models.OriginDataset
	files         []*origin_dataset_file_loader.FileInfo
}

var queries = make(map[string]string)

func NewMetadataSubmitterDatabase(ctx context.Context) (origin_dataset_file_loader.OriginDatasetFileLoader, error) {
	ctx, cancel := context.WithCancel(ctx)

	return &metadataSubmitterClient{
		client: http.DefaultClient,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

func (mdc *metadataSubmitterClient) Ping(ctx context.Context) error {
	if mdc == nil {
		return errors.New("metadata submitter client not initialized")
	}
	rsp, err := mdc.client.Get(path.Join(metadataSubmitterURL, "health"))
	if err != nil {
		return err
	}
	defer func() {
		_ = rsp.Body.Close()
	}()

	if rsp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: %s", rsp.Status)
	}

	return nil
}

func (mdc *metadataSubmitterClient) cacheCleanTicker() {
	var done bool
	ticker := time.NewTicker(30 * time.Minute)
	for {
		select {
		case <-ticker.C:
		case <-mdc.ctx.Done():
			done = true
		}

		if done {
			break
		}

		cacheTTL := time.Minute * 30
		mdc.Lock()
		for k, v := range mdc.cache {
			if v.lastAccessed.Add(cacheTTL).Before(time.Now()) {
				delete(mdc.cache, k)
			}
		}
		mdc.Unlock()
	}
}

// Close terminates the connection to the database
func (mdc *metadataSubmitterClient) Close() error {
	if mdc != nil {
		mdc.cancel()
	}

	return nil
}

func (mdc *metadataSubmitterClient) getDatasetMetadata(ctx context.Context, datasetAccession string) (*cacheEntry, error) {
	mdc.RLock()
	defer mdc.RUnlock()

	entry, ok := mdc.cache[datasetAccession]
	if ok {
		entry.lastAccessed = time.Now()

		return entry, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path.Join(metadataSubmitterURL, "sync", datasetAccession), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build http request: %w", err)
	}
	token, err := mdc.generateAndSignToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate and sign token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rsp, err := mdc.client.Do(req)
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

	entry = &cacheEntry{
		lastAccessed: time.Now(),
		originDataset: &models.OriginDataset{
			Accession: datasetAccession,
		},
		files: nil,
	}

	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s from zip reader: %w", f.Name, err)
		}

		xmlDecoder := xml.NewDecoder(rc)
		switch f.Name {
		case "dataset.xml":
			err = xmlDecoder.Decode(&entry.originDataset.Dataset)
		case "image.xml":
			err = xmlDecoder.Decode(&entry.originDataset.Image)
		case "annotation.xml":
			err = xmlDecoder.Decode(&entry.originDataset.Annotation)
		case "observation.xml":
			err = xmlDecoder.Decode(&entry.originDataset.Observation)
		case "observer.xml":
			err = xmlDecoder.Decode(&entry.originDataset.Observer)
		case "policy.xml":
			err = xmlDecoder.Decode(&entry.originDataset.Policy)
		case "sample.xml":
			err = xmlDecoder.Decode(&entry.originDataset.Sample)
		case "staining.xml":
			err = xmlDecoder.Decode(&entry.originDataset.Staining)
		case "rems.xml":
			var remsEntry metadata_models.Rems
			if err = xmlDecoder.Decode(&remsEntry); err != nil {
				break
			}
			workflowID, err := strconv.Atoi(remsEntry.WorkflowId)
			if err != nil {
				return nil, fmt.Errorf("failed to parse rems workflow id to an integer: %w", err)
			}
			entry.originDataset.RemsWorkflowID = workflowID
			entry.originDataset.RemsOrganisationID = remsEntry.OrganisationId
		case "files.json":
			type file struct {
				FileID string `json:"file_id"`
				Path   string `json:"path"`
			}
			var files []*file

			if err = xmlDecoder.Decode(&files); err != nil {
				break
			}
			for _, f := range files {
				entry.files = append(entry.files, &origin_dataset_file_loader.FileInfo{
					Accession:        f.FileID,
					Path:             f.Path,
					DatasetAccession: datasetAccession,
					MetadataFileType: metadataTypeFromPath(f.Path),
				})
			}

		default:
		}

		_ = rc.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal file %s: %w", f.Name, err)
		}
	}

	mdc.cache[datasetAccession] = entry

	return entry, nil
}

func metadataTypeFromPath(filePath string) metadata_models.MetadataFileType {
	switch {
	case strings.HasSuffix(filePath, "METADATA/dataset.xml.c4gh"):
		return metadata_models.MetadataFileTypeDataset
	case strings.HasSuffix(filePath, "METADATA/image.xml.c4gh"):
		return metadata_models.MetadataFileTypeImage
	case strings.HasSuffix(filePath, "METADATA/annotation.xml.c4gh"):
		return metadata_models.MetadataFileTypeAnnotation
	case strings.HasSuffix(filePath, "METADATA/observation.xml.c4gh"):
		return metadata_models.MetadataFileTypeObservation
	case strings.HasSuffix(filePath, "METADATA/observer.xml.c4gh"):
		return metadata_models.MetadataFileTypeObserver
	case strings.HasSuffix(filePath, "METADATA/policy.xml.c4gh"):
		return metadata_models.MetadataFileTypePolicy
	case strings.HasSuffix(filePath, "METADATA/sample.xml.c4gh"):
		return metadata_models.MetadataFileTypeSample
	case strings.HasSuffix(filePath, "METADATA/staining.xml.c4gh"):
		return metadata_models.MetadataFileTypeStaining
	default:
		return metadata_models.MetadataFileTypeInvalid
	}
}

func (mdc *metadataSubmitterClient) generateAndSignToken() (string, error) {
	claims := map[string]any{
		jwt.ExpirationKey: time.Now().UTC().Add(15 * time.Second),
		jwt.IssuedAtKey:   time.Now().UTC(),
		jwt.IssuerKey:     "bp-dod-sda-gateway",
		jwt.SubjectKey:    "bp-dod-sda-gateway",
		jwt.AudienceKey:   "metadata-submitter",
	}

	jwtKey, err := jwk.ParseKey(privateKey, jwk.WithPEM(true))
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

	tokenString, err := jwt.Sign(token, jwt.WithKey(jwa.ES256, jwtKey))
	if err != nil {
		return "", err
	}

	return string(tokenString), nil
}

func (mdc *metadataSubmitterClient) GetRemsWorkFlowIDAndOrganisationID(ctx context.Context, datasetAccession string) (int, string, error) {
	entry, err := mdc.getDatasetMetadata(ctx, datasetAccession)
	if err != nil {
		return 0, "", err
	}

	return entry.originDataset.RemsWorkflowID, entry.originDataset.RemsOrganisationID, nil
}

func (mdc *metadataSubmitterClient) UnmarshalFileToXml(ctx context.Context, file *origin_dataset_file_loader.FileInfo, dst any) error {
	entry, err := mdc.getDatasetMetadata(ctx, file.DatasetAccession)
	if err != nil {
		return err
	}

	switch file.MetadataFileType {
	case metadata_models.MetadataFileTypeDataset:
		*dst.(*metadata_models.DatasetSet) = *entry.originDataset.Dataset
	case metadata_models.MetadataFileTypeImage:
		*dst.(*metadata_models.ImageSet) = *entry.originDataset.Image
	case metadata_models.MetadataFileTypeAnnotation:
		*dst.(*metadata_models.AnnotationSet) = *entry.originDataset.Annotation
	case metadata_models.MetadataFileTypeObservation:
		*dst.(*metadata_models.ObservationSet) = *entry.originDataset.Observation
	case metadata_models.MetadataFileTypeObserver:
		*dst.(*metadata_models.ObserverSet) = *entry.originDataset.Observer
	case metadata_models.MetadataFileTypePolicy:
		*dst.(*metadata_models.PolicySet) = *entry.originDataset.Policy
	case metadata_models.MetadataFileTypeSample:
		*dst.(*metadata_models.SampleSet) = *entry.originDataset.Sample
	case metadata_models.MetadataFileTypeStaining:
		*dst.(*metadata_models.StainingSet) = *entry.originDataset.Staining
	default:
		return errors.New("unknown metadata file type")
	}

	return nil
}

func (mdc *metadataSubmitterClient) ListDatasetFiles(ctx context.Context, datasetAccession string) ([]*origin_dataset_file_loader.FileInfo, error) {
	entry, err := mdc.getDatasetMetadata(ctx, datasetAccession)
	if err != nil {
		return nil, err
	}

	return entry.files, nil
}
