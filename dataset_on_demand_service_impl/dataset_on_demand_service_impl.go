package dataset_on_demand_service_impl

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"

	"connectrpc.com/connect"
	"github.com/NBISweden/bp-dod-sda-gateway/database"
	dodservice "github.com/NBISweden/bp-dod-sda-gateway/grpc_gen/go/v1"
	dodserviceconnect "github.com/NBISweden/bp-dod-sda-gateway/grpc_gen/go/v1/v1connect"
	"github.com/NBISweden/bp-dod-sda-gateway/models"
	"github.com/NBISweden/bp-dod-sda-gateway/models/metadata_models"
	"github.com/NBISweden/bp-dod-sda-gateway/on_demand_dataset_metadata_file_handler"
	"github.com/NBISweden/bp-dod-sda-gateway/origin_dataset_file_loader"
	"github.com/NBISweden/bp-dod-sda-gateway/pkg/observability"
	"go.opentelemetry.io/otel/attribute"
)

type dodServiceImpl struct {
	originDatasetFileLoader origin_dataset_file_loader.OriginDatasetFileLoader
}

func NewDodServiceImpl(options ...func(*dodServiceImpl)) (dodserviceconnect.DatasetOnDemandServiceHandler, error) {
	serviceImpl := &dodServiceImpl{}
	for _, o := range options {
		o(serviceImpl)
	}

	if serviceImpl.originDatasetFileLoader == nil {
		return nil, errors.New("originDatasetFileLoader is required")
	}

	return serviceImpl, nil
}

func (d *dodServiceImpl) NewOriginDataset(ctx context.Context, c *connect.Request[dodservice.NewOriginDatasetRequest]) (*connect.Response[dodservice.NewOriginDatasetResponse], error) {
	ctx, span := observability.StartSpan(ctx, "NewOriginDataset", attribute.String("dataset-accession", c.Msg.GetDatasetAccession()))
	defer span.End()

	if c.Msg.GetDatasetAccession() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("dataset accession must be provided"))
	}

	fileAccessions := make(map[string]string)

	datasetFiles, err := d.originDatasetFileLoader.ListDatasetFiles(ctx, c.Msg.GetDatasetAccession())
	if err != nil {
		span.Error("failed list dataset files", err)

		return nil, connect.NewError(connect.CodeInternal, nil)
	}
	if len(datasetFiles) == 0 {
		span.Error("no dataset files found", nil)

		return nil, connect.NewError(connect.CodeNotFound, nil)
	}

	originDataset := &models.OriginDataset{
		Accession:          c.Msg.GetDatasetAccession(),
		RemsWorkflowID:     -1, // populated from the rems.xml
		RemsOrganisationID: "", // populated from the rems.xml
		Dataset:            nil,
		Image:              nil,
		Annotation:         nil, // Optional
		Observation:        nil,
		Observer:           nil, // Optional
		Policy:             nil,
		Sample:             nil,
		Staining:           nil,
	}

	// Download metadata files, and extract file accession ids
	for _, file := range datasetFiles {
		var err error
		switch file.MetadataFileType {
		case metadata_models.MetadataFileTypeDataset:
			originDataset.Dataset = new(metadata_models.DatasetSet)
			err = d.originDatasetFileLoader.UnmarshalFileToXml(ctx, file, originDataset.Dataset)
		case metadata_models.MetadataFileTypeImage:
			originDataset.Image = new(metadata_models.ImageSet)
			err = d.originDatasetFileLoader.UnmarshalFileToXml(ctx, file, originDataset.Image)
		case metadata_models.MetadataFileTypeAnnotation:
			originDataset.Annotation = new(metadata_models.AnnotationSet)
			err = d.originDatasetFileLoader.UnmarshalFileToXml(ctx, file, originDataset.Annotation)
		case metadata_models.MetadataFileTypeObservation:
			originDataset.Observation = new(metadata_models.ObservationSet)
			err = d.originDatasetFileLoader.UnmarshalFileToXml(ctx, file, originDataset.Observation)
		case metadata_models.MetadataFileTypeObserver:
			originDataset.Observer = new(metadata_models.ObserverSet)
			err = d.originDatasetFileLoader.UnmarshalFileToXml(ctx, file, originDataset.Observer)
		case metadata_models.MetadataFileTypePolicy:
			originDataset.Policy = new(metadata_models.PolicySet)
			err = d.originDatasetFileLoader.UnmarshalFileToXml(ctx, file, originDataset.Policy)
		case metadata_models.MetadataFileTypeSample:
			originDataset.Sample = new(metadata_models.SampleSet)
			err = d.originDatasetFileLoader.UnmarshalFileToXml(ctx, file, originDataset.Sample)
		case metadata_models.MetadataFileTypeStaining:
			originDataset.Staining = new(metadata_models.StainingSet)
			err = d.originDatasetFileLoader.UnmarshalFileToXml(ctx, file, originDataset.Staining)
		default:
			fileAccessions[file.Path] = file.Accession
		}
		if err != nil {
			span.Error("failed UnmarshalFileToXml file", err, slog.String("filepath", file.Path))

			return nil, connect.NewError(connect.CodeInternal, nil)
		}
	}

	originDataset.RemsWorkflowID, originDataset.RemsOrganisationID, err = d.originDatasetFileLoader.GetRemsWorkFlowIDAndOrganisationID(ctx, c.Msg.GetDatasetAccession())
	if err != nil {
		span.Error("failed get rems xml", err)

		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	if originDataset.RemsWorkflowID == -1 || originDataset.RemsOrganisationID == "" {
		err := errors.New("rems.xml not found")
		span.Error("rems.xml not found", err)

		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}

	// Check that required xml files are present
	if originDataset.Dataset == nil {
		err := errors.New("dataset.xml not found")
		span.Error("dataset.xml not found", err)

		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if originDataset.Image == nil {
		err := errors.New("image.xml not found")
		span.Error("image.xml not found", err)

		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if originDataset.Observation == nil {
		err := errors.New("observation.xml not found")
		span.Error("observation.xml not found", err)

		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if originDataset.Policy == nil {
		err := errors.New("policy.xml not found")
		span.Error("policy.xml not found", err)

		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if originDataset.Sample == nil {
		err := errors.New("sample.xml not found")
		span.Error("sample.xml not found", err)

		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if originDataset.Staining == nil {
		err := errors.New("staining.xml not found")
		span.Error("staining.xml not found", err)

		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}

	tx, err := database.BeginTransaction(ctx)
	if err != nil {
		span.Error("failed to begin database transaction", err)

		return nil, connect.NewError(connect.CodeInternal, nil)
	}
	defer func() {
		if err := tx.Rollback(); err != nil {
			span.Error("failed to rollback database transaction", err)
		}
	}()

	if err := tx.InsertOriginDataset(ctx, originDataset); err != nil {
		span.Error("failed to insert dataset to database", err, slog.String("dataset-accession", originDataset.Accession))

		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	for _, image := range originDataset.Image.Images {
		if image.Accession == "" {
			span.Error("image does not have an accession id", errors.New("image does not have an accession id"), slog.String("image-alias", image.Alias))

			return nil, connect.NewError(connect.CodeFailedPrecondition, nil)
		}

		if err := tx.InsertDatasetImage(ctx, originDataset.Accession, image.Accession); err != nil {
			span.Error("failed to insert dataset image to database", err, slog.String("image-accession", image.Accession))

			return nil, connect.NewError(connect.CodeInternal, nil)
		}
		for _, file := range image.Files.Files {
			var found bool
			for downloadPath, fileAccession := range fileAccessions {
				if strings.HasSuffix(strings.TrimSuffix(downloadPath, ".c4gh"), strings.TrimSuffix(file.Filename, ".c4gh")) {
					delete(fileAccessions, downloadPath)
					found = true
					if err := tx.InsertImageFile(ctx, originDataset.Accession, image.Accession, fileAccession, filepath.Base(file.Filename)); err != nil {
						span.Error("failed to insert image file to database", err, slog.String("image-accession", image.Accession), slog.String("file-accession", fileAccession))

						return nil, connect.NewError(connect.CodeInternal, nil)
					}

					break
				}
			}
			if !found {
				return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("accession id for image file: %s not found", file.Filename))
			}
		}
	}

	if err := tx.Commit(); err != nil {
		span.Error("failed to commit database transaction", err)

		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	return connect.NewResponse(&dodservice.NewOriginDatasetResponse{}), nil
}

func (d *dodServiceImpl) RequestDatasetCreation(ctx context.Context, c *connect.Request[dodservice.RequestDatasetCreationRequest]) (*connect.Response[dodservice.RequestDatasetCreationResponse], error) {
	ctx, span := observability.StartSpan(ctx, "RequestDatasetCreation", attribute.String("user", c.Msg.GetUser()))
	defer span.End()

	if len(c.Msg.GetImageAccessions()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("no image accessions requested"))
	}
	if c.Msg.GetUser() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("user requesting to create on demand dataset must be provided"))
	}

	// calculate the hash of the requested image accession to check for already existing on demand dataset with the same set of images
	imageAccessionHash := hashImageAccessions(c.Msg.GetImageAccessions())
	// Possible future improvement, lock by the imageAccessionHash to avoid race condition when multiple concurrent requests to create same dataset

	existingOnDemandDatasetAccession, err := database.GetOnDemandDatasetAccessionFromImageAccessionsHash(ctx, imageAccessionHash)
	if err != nil {
		span.Error("failed to check for existing on demand dataset image accessions hash", err)

		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	if existingOnDemandDatasetAccession != "" {
		return connect.NewResponse(&dodservice.RequestDatasetCreationResponse{
			OnDemandDatasetAccession: existingOnDemandDatasetAccession,
		}), nil
	}

	originDatasetImages := make(map[string]map[string]struct{})

	// Create a new context without cancel such that if user cancels the request we still proceed to finish the on demand dataset registration
	// This is to avoid scenarios where caller cancels the request while uploading and triggering ingestion for the metadata files, and where it could end up in a partial state
	ctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	defer cancel()

	tx, err := database.BeginTransaction(ctx)
	if err != nil {
		span.Error("failed to begin database transaction", err)

		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	defer func() {
		if err := tx.Rollback(); err != nil {
			span.Error("failed to rollback database transaction", err)
		}
	}()

	for _, imageAccession := range c.Msg.GetImageAccessions() {
		originDatasetAccession, err := tx.GetOriginDatasetAccessionFromImageAccession(ctx, imageAccession)
		if err != nil {
			span.Error("failed to get origin dataset accession from image accession", err, slog.String("image-accession", imageAccession))

			return nil, connect.NewError(connect.CodeInternal, nil)
		}

		if originDatasetAccession == "" {
			span.Warn("no dataset found from image accession", slog.String("image-accession", imageAccession))

			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		if _, ok := originDatasetImages[originDatasetAccession]; !ok {
			originDatasetImages[originDatasetAccession] = map[string]struct{}{imageAccession: {}}

			continue
		}

		if _, ok := originDatasetImages[originDatasetAccession][imageAccession]; ok {
			span.Info("user requested to combine identical image accessions", slog.String("image-accession", imageAccession))

			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("duplicate image accession requested"))
		}

		originDatasetImages[originDatasetAccession][imageAccession] = struct{}{}
	}

	originDatasets := make(map[string]*models.OriginDataset)

	var ensureSameWorkflowID int

	var ensureSameTou *metadata_models.PolicySet

	for originDatasetAccession := range originDatasetImages {
		originDataset, err := tx.GetOriginDataset(ctx, originDatasetAccession)
		if err != nil {
			span.Error("failed to get origin dataset", err, slog.String("accession", originDatasetAccession))

			return nil, connect.NewError(connect.CodeInternal, nil)
		}

		if originDataset == nil {
			span.Error("failed to find origin dataset", errors.New("failed to find origin dataset"), slog.String("accession", originDatasetAccession))

			return nil, connect.NewError(connect.CodeInternal, nil)
		}

		originDatasets[originDatasetAccession] = originDataset

		if ensureSameWorkflowID == 0 {
			ensureSameWorkflowID = originDataset.RemsWorkflowID
		}

		if ensureSameTou == nil {
			ensureSameTou = originDataset.Policy
		}

		// If only images from one dataset no need to compare
		if len(originDatasetImages) == 1 {
			continue
		}

		if ensureSameWorkflowID != originDataset.RemsWorkflowID {
			span.Info("user tried to combine datasets with different workflows id")

			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("can not combine images from datasets originating from different DaCs"))
		}

		if !ensureSameTou.Equal(originDataset.Policy) {
			span.Info("user tried to combine datasets with different terms of use")

			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("can not combine images from datasets with different terms of use"))
		}
	}

	onDemandDataset := buildOnDemandDataset(ctx, originDatasets, originDatasetImages)
	onDemandDataset.RequestedByUser = c.Msg.GetUser()
	span.SetAttributes(attribute.String("on-demand-dataset-accession", onDemandDataset.Accession))

	if err := tx.InsertOnDemandDataset(ctx, onDemandDataset, imageAccessionHash); err != nil {
		span.Error("failed to insert on demand dataset", err)

		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	for originAccession, imageAccessions := range originDatasetImages {
		for imageAccession := range imageAccessions {
			if err := tx.InsertOnDemandDatasetImage(ctx, onDemandDataset.Accession, imageAccession); err != nil {
				span.Error("failed to insert on demand dataset image", err, slog.String("origin-accession", originAccession), slog.String("image-accession", imageAccession))

				return nil, connect.NewError(connect.CodeInternal, nil)
			}
		}
	}

	if err := on_demand_dataset_metadata_file_handler.RegisterOnDemandDataset(ctx, onDemandDataset); err != nil {
		span.Error("failed to register on demand dataset", err)

		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	// For now, we risk failing to commit after having uploaded and triggered ingestion for the metadata files(done by RegisterOnDemandDataset), but should be ok for now
	if err := tx.Commit(); err != nil {
		span.Error("failed to commit database transaction", err)

		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	span.Info("on demand dataset registered successfully")

	return connect.NewResponse(&dodservice.RequestDatasetCreationResponse{
		OnDemandDatasetAccession: onDemandDataset.Accession,
	}), nil
}

func (d *dodServiceImpl) GetOnDemandDatasetStatus(ctx context.Context, c *connect.Request[dodservice.GetOnDemandDatasetStatusRequest]) (*connect.Response[dodservice.GetOnDemandDatasetStatusResponse], error) {
	ctx, span := observability.StartSpan(ctx, "GetOnDemandDatasetStatus", attribute.String("accession", c.Msg.GetOnDemandDatasetAccession()))
	defer span.End()

	if c.Msg.GetOnDemandDatasetAccession() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("empty dod dataset accession"))
	}

	onDemandDatasetReleased, err := database.IsOnDemandDatasetReleased(ctx, c.Msg.GetOnDemandDatasetAccession())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		span.Error("failed to check if on demand dataset is released", err)

		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	res := dodservice.GetOnDemandDatasetStatusResponse{
		Status: dodservice.GetOnDemandDatasetStatusResponse_STATUS_CREATING,
	}

	if onDemandDatasetReleased {
		res.Status = dodservice.GetOnDemandDatasetStatusResponse_STATUS_RELEASED
	}

	return connect.NewResponse(&res), nil
}

// hashImageAccessions takes a slice of image accessions, sorts them, removes duplicates and returns a sha256 hash of the slice
func hashImageAccessions(ids []string) string {
	sorted := append([]string(nil), ids...)
	slices.Sort(sorted)

	n := 0
	for _, id := range sorted {
		if n == 0 || id != sorted[n-1] {
			sorted[n] = id
			n++
		}
	}
	sorted = sorted[:n]

	h := sha256.New()
	for _, id := range sorted {
		_, _ = h.Write([]byte(id))
		_, _ = h.Write([]byte{0}) // separator
	}

	return hex.EncodeToString(h.Sum(nil))
}
