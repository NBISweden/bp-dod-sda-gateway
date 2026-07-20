package on_demand_dataset_metadata_file_handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/NBISweden/bp-dod-sda-gateway/database"
	"github.com/NBISweden/bp-dod-sda-gateway/models"
	"github.com/NBISweden/bp-dod-sda-gateway/models/metadata_models"
	"github.com/NBISweden/bp-dod-sda-gateway/pkg/observability"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/neicnordic/crypt4gh/keys"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/chacha20poly1305"
)

type dodMetadataFileHandler struct {
	ctx context.Context

	httpClient *http.Client

	transferManagerClient *transfermanager.Client
	c4ghPublicKey         [chacha20poly1305.KeySize]byte

	uploadUser string

	sync.Mutex
}

var dmfh *dodMetadataFileHandler

func Init(ctx context.Context) error {
	dmfh = &dodMetadataFileHandler{
		ctx:        ctx,
		httpClient: &http.Client{},
	}

	// Parse the user from the inbox.Token and extract the subject, as this is needed to trigger ingestion and dataset creation
	parser := jwt.NewParser()

	tokenClaims := jwt.MapClaims{}
	_, _, err := parser.ParseUnverified(inboxToken, tokenClaims)
	if err != nil {
		return fmt.Errorf("failed to parse(unverified) the inbox token: %w", err)
	}

	tokenSubject, err := tokenClaims.GetSubject()
	if err != nil || tokenSubject == "" {
		return fmt.Errorf("failed to extract subject (sub) from inbox token: %w", err)
	}

	dmfh.uploadUser = tokenSubject

	pubKey, err := os.Open(filepath.Clean(c4ghPublicKeyFilePath))
	if err != nil {
		return fmt.Errorf("failed to open c4ghPublicKeyFilePath: %w", err)
	}
	defer func() {
		_ = pubKey.Close()
	}()
	dmfh.c4ghPublicKey, err = keys.ReadPublicKey(pubKey)
	if err != nil {
		return fmt.Errorf("failed to read public key from the c4ghPublicKeyFilePath: %w", err)
	}

	awsconf, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			"dummy", // The s3inbox does not care about the access key, it authenticates the session token
			"dummy", // The s3inbox does not care about the secret key, it authenticates the session token
			inboxToken,
		)),
		awsconfig.WithRegion("us-east-1"),
		awsconfig.WithBaseEndpoint(inboxEndpointUrl),
	)
	if err != nil {
		return fmt.Errorf("failed to load aws config, reason %v", err)
	}

	s3Client := s3.NewFromConfig(awsconf, func(o *s3.Options) {
		o.UsePathStyle = true
		o.EndpointOptions.DisableHTTPS = inboxDisableHTTPS
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		o.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
	})

	// Create an datasetCreationRequestHandler with the session and default options
	dmfh.transferManagerClient = transfermanager.New(s3Client)

	datasetsMetadataFiles, err := database.ListDatasetOnDemandMetadataFiles(ctx)
	if err != nil {
		return err
	}

	for datasetAccession, metadataFiles := range datasetsMetadataFiles {
		go dmfh.monitorDatasetMetadataFiles(datasetAccession, metadataFiles)
	}

	return nil
}

func (dmfh *dodMetadataFileHandler) monitorDatasetMetadataFiles(datasetAccession string, datasetMetadataFiles map[metadata_models.MetadataFileType]string) {
	for {
		if dmfh.ctx.Err() != nil {
			slog.Info("on demand dataset metadata file handler - stopped, context canceled")

			break
		}

		done, err := dmfh.pollAndProcess(datasetAccession, datasetMetadataFiles)
		if err != nil {
			slog.Warn("failed to process dataset metadata files", "error", err, "retry-in", sdaAPIPollInterval.String())
		}
		if done {
			slog.Info("on demand dataset metadata file handler - process dataset metadata files finished", "accession", datasetAccession)

			break
		}

		select {
		case <-time.After(sdaAPIPollInterval):
		case <-dmfh.ctx.Done():
			slog.Info("on demand dataset metadata file handler - stopped, context canceled")
			return
		}
	}
}

func (dmfh *dodMetadataFileHandler) pollAndProcess(datasetAccession string, datasetMetadataFiles map[metadata_models.MetadataFileType]string) (bool, error) {
	ctx, span := observability.Tracer().Start(dmfh.ctx, "monitorDatasetMetadataFiles", trace.WithAttributes(attribute.String("accession", datasetAccession)))
	defer span.End()

	// verified -> do accession
	// all ready -> mappings

	metadataFiles, err := dmfh.listMetadataFiles(ctx, datasetAccession)
	if err != nil {
		return false, fmt.Errorf("failed to list metadata files: %w", err)
	}

	allReady := true
	for _, metadataFile := range metadataFiles {
		switch metadataFile.Status {
		// Based on https://github.com/neicnordic/sensitive-data-archive/blob/main/postgresql/initdb.d/01_main.sql#L69
		case "uploaded":
			slog.Debug("metadata file still in uploaded status", "sda-id", metadataFile.FileId)
			allReady = false
		case "registered", "backed up", "downloaded", "error", "disabled", "enabled":
			slog.Warn("unexpected metadata file status", "status", metadataFile.Status, "sda-id", metadataFile.FileId)
			allReady = false
		case "submitted", "ingested", "archived":
			// Keep waiting until "verified"
			allReady = false
		case "verified":
			var fileAccession string
			switch {
			case strings.Contains(metadataFile.InboxPath, metadata_models.MetadataFileTypeDataset.String()):
				fileAccession = datasetMetadataFiles[metadata_models.MetadataFileTypeDataset]
			case strings.Contains(metadataFile.InboxPath, metadata_models.MetadataFileTypeImage.String()):
				fileAccession = datasetMetadataFiles[metadata_models.MetadataFileTypeImage]
			case strings.Contains(metadataFile.InboxPath, metadata_models.MetadataFileTypeObservation.String()):
				fileAccession = datasetMetadataFiles[metadata_models.MetadataFileTypeObservation]
			case strings.Contains(metadataFile.InboxPath, metadata_models.MetadataFileTypeObserver.String()):
				fileAccession = datasetMetadataFiles[metadata_models.MetadataFileTypeObserver]
			case strings.Contains(metadataFile.InboxPath, metadata_models.MetadataFileTypePolicy.String()):
				fileAccession = datasetMetadataFiles[metadata_models.MetadataFileTypePolicy]
			case strings.Contains(metadataFile.InboxPath, metadata_models.MetadataFileTypeSample.String()):
				fileAccession = datasetMetadataFiles[metadata_models.MetadataFileTypeSample]
			case strings.Contains(metadataFile.InboxPath, metadata_models.MetadataFileTypeStaining.String()):
				fileAccession = datasetMetadataFiles[metadata_models.MetadataFileTypeStaining]
			default:
			}

			if fileAccession == "" {
				slog.Warn("missing accession for metadata file", "inbox-path", metadataFile.InboxPath, "sda-id", metadataFile.FileId)
				allReady = false

				continue
			}

			if err := dmfh.triggerFileAccession(ctx, metadataFile.FileId, fileAccession); err != nil {
				return false, fmt.Errorf("failed to trigger file accession: %w", err)
			}
			allReady = false
		case "ready":

		default:
			slog.Warn("unknown metadata file status", "status", metadataFile.Status)
			allReady = false
		}
	}

	if !allReady || len(metadataFiles) == 0 {
		return false, nil
	}

	remsMetadata, err := database.GetOnDemandDatasetRemsMetadata(ctx, datasetAccession)
	if err != nil {
		return false, err
	}

	remsResourceID, err := dmfh.createRemsResource(ctx, remsMetadata, datasetAccession)
	if err != nil {
		return false, fmt.Errorf("failed to create rems resource for on demand dataset: %w", err)
	}
	if err := dmfh.createRemsCatalogueItem(ctx, remsMetadata, datasetAccession, remsResourceID); err != nil {
		return false, fmt.Errorf("failed to create rems catalog item for on demand dataset: %w", err)
	}

	datasetFileAccessions, err := database.ListDatasetOnDemandImageFileAccessions(ctx, datasetAccession)
	if err != nil {
		return false, fmt.Errorf("failed to list on demand dataset image accessions: %w", err)
	}

	for metadataType, datasetFileAccession := range datasetMetadataFiles {
		// Exclude rems
		if metadataType == metadata_models.MetadataFileTypeRems {
			continue
		}
		datasetFileAccessions = append(datasetFileAccessions, datasetFileAccession)
	}

	if err := dmfh.triggerDatasetCreation(ctx, datasetAccession, datasetFileAccessions); err != nil {
		return false, fmt.Errorf("failed to trigger dataset creation: %w", err)
	}

	// Small sleep to allow the dataset creation to be processed, as otherwise sda-api will respond with not found while mapper is processing creation request
	time.Sleep(5 * time.Second)

	if err := dmfh.triggerDatasetRelease(ctx, datasetAccession); err != nil {
		return false, fmt.Errorf("failed to trigger on demand dataset release: %w", err)
	}

	if err := database.SetOnDemandDatasetReleased(ctx, datasetAccession); err != nil {
		return false, fmt.Errorf("failed to set on demand dataset as released: %w", err)
	}

	return true, nil
}

func RegisterOnDemandDataset(ctx context.Context, dodDataset *models.OnDemandDataset) error {
	ctx, span := observability.Tracer().Start(ctx, "RegisterOnDemandDataset", trace.WithAttributes(attribute.String("accession", dodDataset.Accession)))
	defer span.End()

	if err := dmfh.marshalEncryptAndUploadFile(ctx, filepath.Join(dodDataset.Accession, "METADATA", "dataset.xml"), dodDataset.DatasetMetadata.MetadataSet); err != nil {
		slog.Warn("failed to upload dataset xml", "error", err, "accession", dodDataset.Accession)

		return fmt.Errorf("failed to upload dataset xml: %w", err)
	}
	if err := dmfh.marshalEncryptAndUploadFile(ctx, filepath.Join(dodDataset.Accession, "METADATA", "image.xml"), dodDataset.ImageMetadata.MetadataSet); err != nil {
		slog.Warn("failed to upload image xml", "error", err, "accession", dodDataset.Accession)

		return fmt.Errorf("failed to upload image xml: %w", err)
	}
	if err := dmfh.marshalEncryptAndUploadFile(ctx, filepath.Join(dodDataset.Accession, "METADATA", "observation.xml"), dodDataset.ObservationMetadata.MetadataSet); err != nil {
		slog.Warn("failed to upload observation xml", "error", err, "accession", dodDataset.Accession)

		return fmt.Errorf("failed to upload observation xml: %w", err)
	}
	if dodDataset.ObserverMetadata != nil {
		if err := dmfh.marshalEncryptAndUploadFile(ctx, filepath.Join(dodDataset.Accession, "METADATA", "observer.xml"), dodDataset.ObserverMetadata.MetadataSet); err != nil {
			slog.Warn("failed to upload observer xml", "error", err, "accession", dodDataset.Accession)

			return fmt.Errorf("failed to upload observer xml: %w", err)
		}
	}
	if err := dmfh.marshalEncryptAndUploadFile(ctx, filepath.Join(dodDataset.Accession, "METADATA", "policy.xml"), dodDataset.PolicyMetadata.MetadataSet); err != nil {
		slog.Warn("failed to upload policy xml", "error", err, "accession", dodDataset.Accession)

		return fmt.Errorf("failed to upload policy xml: %w", err)
	}
	if err := dmfh.marshalEncryptAndUploadFile(ctx, filepath.Join(dodDataset.Accession, "METADATA", "sample.xml"), dodDataset.SampleMetadata.MetadataSet); err != nil {
		slog.Warn("failed to upload sample xml", "error", err, "accession", dodDataset.Accession)

		return fmt.Errorf("failed to upload sample xml: %w", err)
	}

	if err := dmfh.marshalEncryptAndUploadFile(ctx, filepath.Join(dodDataset.Accession, "METADATA", "staining.xml"), dodDataset.StainingMetadata.MetadataSet); err != nil {
		slog.Warn("failed to upload staining xml", "error", err, "accession", dodDataset.Accession)

		return fmt.Errorf("failed to upload staining xml: %w", err)
	}

	if err := dmfh.triggerFileIngest(ctx, dodDataset.Accession, dodDataset.DatasetMetadata.MetadataFileType()); err != nil {
		slog.Warn("failed to trigger ingestion for dataset xml", "error", err, "accession", dodDataset.Accession)

		return fmt.Errorf("failed to trigger ingestion for dataset xml: %w", err)
	}
	if err := dmfh.triggerFileIngest(ctx, dodDataset.Accession, dodDataset.ImageMetadata.MetadataFileType()); err != nil {
		slog.Warn("failed to trigger ingestion for image xml", "error", err, "accession", dodDataset.Accession)

		return fmt.Errorf("failed to trigger ingestion for image xml: %w", err)
	}
	if err := dmfh.triggerFileIngest(ctx, dodDataset.Accession, dodDataset.ObservationMetadata.MetadataFileType()); err != nil {
		slog.Warn("failed to trigger ingestion for observation xml", "error", err, "accession", dodDataset.Accession)

		return fmt.Errorf("failed to trigger ingestion for observation xml: %w", err)
	}
	if dodDataset.ObserverMetadata != nil {
		if err := dmfh.triggerFileIngest(ctx, dodDataset.Accession, dodDataset.ObserverMetadata.MetadataFileType()); err != nil {
			slog.Warn("failed to trigger ingestion for observer xml", "error", err, "accession", dodDataset.Accession)

			return fmt.Errorf("failed to trigger ingestion for observer xml: %w", err)
		}
	}
	if err := dmfh.triggerFileIngest(ctx, dodDataset.Accession, dodDataset.PolicyMetadata.MetadataFileType()); err != nil {
		slog.Warn("failed to trigger ingestion for policy xml", "error", err, "accession", dodDataset.Accession)

		return fmt.Errorf("failed to trigger ingestion for policy xml: %w", err)
	}
	if err := dmfh.triggerFileIngest(ctx, dodDataset.Accession, dodDataset.SampleMetadata.MetadataFileType()); err != nil {
		slog.Warn("failed to trigger ingestion for sample xml", "error", err, "accession", dodDataset.Accession)

		return fmt.Errorf("failed to trigger ingestion for sample xml: %w", err)
	}
	if err := dmfh.triggerFileIngest(ctx, dodDataset.Accession, dodDataset.StainingMetadata.MetadataFileType()); err != nil {
		slog.Warn("failed to trigger ingestion for staining xml", "error", err, "accession", dodDataset.Accession)

		return fmt.Errorf("failed to trigger ingestion for staining xml: %w", err)
	}

	metadataFiles := map[metadata_models.MetadataFileType]string{
		dodDataset.DatasetMetadata.MetadataSet.MetadataFileType():     dodDataset.DatasetMetadata.Accession,
		dodDataset.ImageMetadata.MetadataSet.MetadataFileType():       dodDataset.ImageMetadata.Accession,
		dodDataset.ObservationMetadata.MetadataSet.MetadataFileType(): dodDataset.ObservationMetadata.Accession,
		dodDataset.PolicyMetadata.MetadataSet.MetadataFileType():      dodDataset.PolicyMetadata.Accession,
		dodDataset.SampleMetadata.MetadataSet.MetadataFileType():      dodDataset.SampleMetadata.Accession,
		dodDataset.StainingMetadata.MetadataSet.MetadataFileType():    dodDataset.StainingMetadata.Accession,
	}
	if dodDataset.ObserverMetadata != nil {
		metadataFiles[dodDataset.ObserverMetadata.MetadataSet.MetadataFileType()] = dodDataset.ObserverMetadata.Accession
	}

	go dmfh.monitorDatasetMetadataFiles(dodDataset.Accession, metadataFiles)

	return nil
}

func (dmfh *dodMetadataFileHandler) triggerFileIngest(ctx context.Context, datasetAccession string, metadataFileType metadata_models.MetadataFileType) error {
	ctx, span := observability.Tracer().Start(ctx, "triggerFileIngest", trace.WithAttributes(attribute.String("sda-api-url", sdaAPIUrl), attribute.String("metadata-file-type", metadataFileType.String())))
	defer span.End()

	endpoint, err := url.JoinPath(sdaAPIUrl, "file", "ingest")
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}

	ingestReq := struct {
		FilePath string `json:"filepath"`
		User     string `json:"user"`
	}{
		FilePath: fmt.Sprintf("%s/METADATA/%s.xml.c4gh", datasetAccession, metadataFileType.String()),
		User:     dmfh.uploadUser,
	}

	reqBody, err := json.Marshal(ingestReq)
	if err != nil {
		return fmt.Errorf("failed to marshal ingest body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+inboxToken)

	resp, err := dmfh.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (dmfh *dodMetadataFileHandler) triggerFileAccession(ctx context.Context, fileID, fileAccession string) error {
	ctx, span := observability.Tracer().Start(ctx, "triggerFileAccession", trace.WithAttributes(attribute.String("sda-api-url", sdaAPIUrl)))
	defer span.End()

	endpoint, err := url.JoinPath(sdaAPIUrl, "file", "accession")
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}
	query := url.Values{}
	query.Set("fileid", fileID)
	query.Set("accessionid", fileAccession)

	if enc := query.Encode(); enc != "" {
		endpoint += "?" + enc
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+inboxToken)

	resp, err := dmfh.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (dmfh *dodMetadataFileHandler) triggerDatasetCreation(ctx context.Context, datasetAccession string, fileAccessions []string) error {
	ctx, span := observability.Tracer().Start(ctx, "triggerDatasetCreation", trace.WithAttributes(attribute.String("sda-api-url", sdaAPIUrl), attribute.String("accession", datasetAccession)))
	defer span.End()

	endpoint, err := url.JoinPath(sdaAPIUrl, "dataset", "create")
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}

	datasetCreateReq := struct {
		DatasetAccession string   `json:"dataset_id"`
		FileAccessionIDs []string `json:"accession_ids"`
		User             string   `json:"user"`
	}{
		DatasetAccession: datasetAccession,
		FileAccessionIDs: fileAccessions,
		User:             dmfh.uploadUser,
	}

	reqBody, err := json.Marshal(datasetCreateReq)
	if err != nil {
		return fmt.Errorf("failed to marshal dataset request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+inboxToken)

	resp, err := dmfh.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (dmfh *dodMetadataFileHandler) triggerDatasetRelease(ctx context.Context, datasetAccession string) error {
	ctx, span := observability.Tracer().Start(ctx, "triggerDatasetRelease", trace.WithAttributes(attribute.String("sda-api-url", sdaAPIUrl), attribute.String("accession", datasetAccession)))
	defer span.End()

	endpoint, err := url.JoinPath(sdaAPIUrl, "dataset", "release", datasetAccession)
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+inboxToken)

	resp, err := dmfh.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (dmfh *dodMetadataFileHandler) listMetadataFiles(ctx context.Context, datasetAccession string) ([]*fileInfo, error) {
	ctx, span := observability.Tracer().Start(ctx, "listMetadataFiles", trace.WithAttributes(attribute.String("sda-api-url", sdaAPIUrl)))
	defer span.End()

	endpoint, err := url.JoinPath(sdaAPIUrl, "files")
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	return paginate(ctx, func(ctx context.Context, nextCursor string) ([]*fileInfo, string, error) {
		listUrl := endpoint
		query := url.Values{}
		query.Set("path_prefix", datasetAccession)
		if nextCursor != "" {
			query.Set("cursor", nextCursor)
		}
		if enc := query.Encode(); enc != "" {
			listUrl += "?" + enc
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, listUrl, nil)
		if err != nil {
			return nil, "", fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer "+inboxToken)

		resp, err := dmfh.httpClient.Do(req)
		if err != nil {
			return nil, "", fmt.Errorf("http request: %w", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)

			return nil, "", fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
		}

		var fileList []*fileInfo
		if err := json.NewDecoder(resp.Body).Decode(&fileList); err != nil {
			return nil, "", fmt.Errorf("failed to decode /files response: %w", err)
		}

		return fileList, resp.Header.Get("X-Next-Cursor"), nil
	})
}
