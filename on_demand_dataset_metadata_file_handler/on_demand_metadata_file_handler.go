package on_demand_dataset_metadata_file_handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
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

type DodMetadataFileHandler interface {
	RegisterOnDemandDataset(ctx context.Context, dodDataset *models.OnDemandDataset) error
}

var dmfh DodMetadataFileHandler

func RegisterDodMetadataFileHandler(newDmfh DodMetadataFileHandler) {
	dmfh = newDmfh
}
func RegisterOnDemandDataset(ctx context.Context, dodDataset *models.OnDemandDataset) error {
	return dmfh.RegisterOnDemandDataset(ctx, dodDataset)
}

type dodMetadataFileHandler struct {
	ctx context.Context

	httpClient *http.Client

	transferManagerClient *transfermanager.Client
	c4ghPublicKey         [chacha20poly1305.KeySize]byte

	uploadUser string

	sync.Mutex
}

func Init(ctx context.Context) error {
	if dmfh != nil {
		return errors.New("dod metadata file handler has already been initialized")
	}

	newDmfh := &dodMetadataFileHandler{
		ctx:        ctx,
		httpClient: &http.Client{},
	}

	// Parse the user from the inbox.Token and extract the subject, as this is needed to trigger ingestion and dataset creation
	parser := jwt.NewParser()

	tokenClaims := jwt.MapClaims{}
	_, _, err := parser.ParseUnverified(inboxToken, tokenClaims)
	if err != nil {
		return fmt.Errorf("failed to parse the inbox token: %w", err)
	}

	tokenSubject, err := tokenClaims.GetSubject()
	if err != nil || tokenSubject == "" {
		return fmt.Errorf("failed to extract subject (sub) from inbox token: %w", err)
	}

	newDmfh.uploadUser = tokenSubject

	pubKey, err := os.Open(filepath.Clean(c4ghPublicKeyFilePath))
	if err != nil {
		return fmt.Errorf("failed to open c4ghPublicKeyFilePath: %w", err)
	}
	defer func() {
		_ = pubKey.Close()
	}()
	newDmfh.c4ghPublicKey, err = keys.ReadPublicKey(pubKey)
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
	newDmfh.transferManagerClient = transfermanager.New(s3Client)

	datasetsMetadataFiles, err := database.ListUnreleasedOnDemandDatasetMetadataFiles(ctx)
	if err != nil {
		return err
	}

	for datasetAccession, metadataFiles := range datasetsMetadataFiles {
		go newDmfh.monitorDatasetMetadataFiles(datasetAccession, metadataFiles)
	}

	dmfh = newDmfh
	return nil
}

func (dmfh *dodMetadataFileHandler) RegisterOnDemandDataset(ctx context.Context, dodDataset *models.OnDemandDataset) error {
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
			slog.Info("on demand dataset metadata file handler - stopped, context done")

			return
		}
	}
}

func (dmfh *dodMetadataFileHandler) pollAndProcess(datasetAccession string, datasetMetadataFiles map[metadata_models.MetadataFileType]string) (bool, error) {
	ctx, span := observability.Tracer().Start(dmfh.ctx, "pollAndProcess", trace.WithAttributes(attribute.String("accession", datasetAccession)))
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

	imageBaseFileNames, err := database.ListOnDemandDatasetImageFileAccessions(ctx, datasetAccession)
	if err != nil {
		return false, fmt.Errorf("failed to list on demand dataset image accessions: %w", err)
	}

	if err := dmfh.triggerDatasetCreation(ctx, datasetAccession, datasetMetadataFiles, imageBaseFileNames); err != nil {
		return false, fmt.Errorf("failed to trigger dataset creation: %w", err)
	}

	// Small sleep to allow the dataset creation to be processed, as otherwise sda-api will respond with not found while mapper is processing creation request on the dataset release request
	time.Sleep(5 * time.Second)

	if err := dmfh.triggerDatasetRelease(ctx, datasetAccession); err != nil {
		return false, fmt.Errorf("failed to trigger on demand dataset release: %w", err)
	}

	if err := database.SetOnDemandDatasetReleased(ctx, datasetAccession); err != nil {
		return false, fmt.Errorf("failed to set on demand dataset as released: %w", err)
	}

	return true, nil
}
