package dataset_on_demand_service_impl

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
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
	"go.opentelemetry.io/otel/trace"
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
	ctx, span := observability.Tracer().Start(ctx, "NewOriginDataset", trace.WithAttributes(attribute.String("dataset-accession", c.Msg.GetDatasetAccession())))
	defer span.End()

	if c.Msg.GetDatasetAccession() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("dataset accession must be provided"))
	}

	fileAccessions := make(map[string]string)

	datasetFiles, err := d.originDatasetFileLoader.ListDatasetFiles(ctx, c.Msg.GetDatasetAccession())
	if err != nil {
		slog.Warn("failed list dataset files", "error", err)

		return nil, connect.NewError(connect.CodeInternal, nil)
	}
	if len(datasetFiles) == 0 {
		slog.Warn("no dataset files found", "error", err, "dataset-accession", c.Msg.GetDatasetAccession())

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
			slog.Warn("failed UnmarshalFileToXml file", "error", err, "dataset-accession", originDataset.Accession, "filepath", file.Path)

			return nil, connect.NewError(connect.CodeInternal, nil)
		}
	}

	originDataset.RemsWorkflowID, originDataset.RemsOrganisationID, err = d.originDatasetFileLoader.GetRemsWorkFlowIDAndOrganisationID(ctx, c.Msg.GetDatasetAccession())
	if err != nil {
		slog.Warn("failed get rems xml", "error", err, "dataset-accession", originDataset.Accession)

		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	if originDataset.RemsWorkflowID == -1 || originDataset.RemsOrganisationID == "" {
		slog.Warn("rems.xml not found", "dataset-accession", originDataset.Accession)

		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("rems.xml not found"))
	}

	// Check that required xml files are present
	if originDataset.Dataset == nil {
		slog.Warn("dataset.xml not found", "dataset-accession", originDataset.Accession)

		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("dataset.xml not found"))
	}
	if originDataset.Image == nil {
		slog.Warn("image.xml not found", "dataset-accession", originDataset.Accession)

		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("image.xml not found"))
	}
	if originDataset.Observation == nil {
		slog.Warn("observation.xml not found", "dataset-accession", originDataset.Accession)

		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("observation.xml not found"))
	}
	if originDataset.Policy == nil {
		slog.Warn("policy.xml not found", "dataset-accession", originDataset.Accession)

		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("policy.xml not found"))
	}
	if originDataset.Sample == nil {
		slog.Warn("sample.xml not found", "dataset-accession", originDataset.Accession)

		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("sample.xml not found"))
	}
	if originDataset.Staining == nil {
		slog.Warn("staining.xml not found", "dataset-accession", originDataset.Accession)

		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("staining.xml not found"))
	}

	tx, err := database.BeginTransaction(ctx)
	if err != nil {
		slog.Warn("failed to begin database transaction", "error", err)

		return nil, connect.NewError(connect.CodeInternal, nil)
	}
	defer func() {
		if err := tx.Rollback(); err != nil {
			slog.Warn("failed to rollback database transaction", "error", err)
		}
	}()

	if err := tx.InsertOriginDataset(ctx, originDataset); err != nil {
		slog.Warn("failed to insert dataset to database", "error", err, "dataset-accession", originDataset.Accession)

		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	for _, image := range originDataset.Image.Images {
		if image.Accession == "" {
			slog.Warn("image does not have an accession id", "error", err, "dataset-accession", originDataset.Accession, "image-accession", image.Accession)

			return nil, connect.NewError(connect.CodeFailedPrecondition, nil)
		}

		if err := tx.InsertDatasetImage(ctx, originDataset.Accession, image.Accession); err != nil {
			slog.Warn("failed to insert dataset image to database", "error", err, "dataset-accession", originDataset.Accession, "image-accession", image.Accession)

			return nil, connect.NewError(connect.CodeInternal, nil)
		}
		for _, file := range image.Files.Files {
			var found bool
			for downloadPath, fileAccession := range fileAccessions {
				if strings.HasSuffix(strings.TrimSuffix(downloadPath, ".c4gh"), strings.TrimSuffix(file.Filename, ".c4gh")) {
					delete(fileAccessions, downloadPath)
					found = true
					if err := tx.InsertImageFile(ctx, originDataset.Accession, image.Accession, fileAccession, filepath.Base(file.Filename)); err != nil {
						slog.Warn("failed to insert image file to database", "error", err, "dataset-accession", originDataset.Accession, "image-accession", image.Accession, "file-accession", fileAccession)

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
		slog.Warn("failed to commit database transaction", "error", err)

		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	return connect.NewResponse(&dodservice.NewOriginDatasetResponse{}), nil
}

func (d *dodServiceImpl) RequestDatasetCreation(ctx context.Context, c *connect.Request[dodservice.RequestDatasetCreationRequest]) (*connect.Response[dodservice.RequestDatasetCreationResponse], error) {
	ctx, span := observability.Tracer().Start(ctx, "RequestDatasetCreation")
	defer span.End()

	if len(c.Msg.GetImageAccessions()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("no image accessions requested"))
	}
	if c.Msg.GetUser() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("user requesting to create on demand dataset must be provided"))
	}

	originDatasetImages := make(map[string][]string)

	tx, err := database.BeginTransaction(ctx)
	if err != nil {
		slog.Warn("failed to begin database transaction", "error", err)

		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	defer func() {
		if err := tx.Rollback(); err != nil {
			slog.Warn("failed to rollback database transaction", "error", err)
		}
	}()

	for _, imageAccession := range c.Msg.GetImageAccessions() {
		originDatasetAccession, err := tx.GetOriginDatasetAccessionFromImageAccession(ctx, imageAccession)
		if err != nil {
			slog.Warn("failed to get origin dataset accession from image accession", "error", err, "image-accession", imageAccession)

			return nil, connect.NewError(connect.CodeInternal, nil)
		}

		if originDatasetAccession == "" {
			slog.Info("no dataset found from image accession", "image-accession", imageAccession)

			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		if originDatasetImages[originDatasetAccession] == nil {
			originDatasetImages[originDatasetAccession] = []string{imageAccession}

			continue
		}

		originDatasetImages[originDatasetAccession] = append(originDatasetImages[originDatasetAccession], imageAccession)
	}

	originDatasets := make(map[string]*models.OriginDataset)

	var ensureSameWorkflowID int

	var ensureSameTou *metadata_models.PolicySet

	for originDatasetAccession := range originDatasetImages {
		originDataset, err := tx.GetOriginDataset(ctx, originDatasetAccession)
		if err != nil {
			slog.Warn("failed to get origin dataset", "error", err, "accession", originDatasetAccession)

			return nil, connect.NewError(connect.CodeInternal, nil)
		}

		if originDataset == nil {
			slog.Error("failed to find origin dataset", "accession", originDatasetAccession)

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
			slog.Info("user tried to combine datasets with different workflows id")

			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("can not combine images from datasets originating from different DaCs"))
		}

		if !ensureSameTou.Equal(originDataset.Policy) {
			slog.Info("user tried to combine datasets with different terms of use")

			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("can not combine images from datasets with different terms of use"))
		}
	}

	onDemandDataset := buildOnDemandDataset(ctx, originDatasets, originDatasetImages)
	onDemandDataset.RequestedByUser = c.Msg.GetUser()

	if err := tx.InsertOnDemandDataset(ctx, onDemandDataset); err != nil {
		slog.Warn("failed to insert on demand dataset", "error", err)

		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	for originAccession, imageAccessions := range originDatasetImages {
		for _, imageAccession := range imageAccessions {
			if err := tx.InsertOnDemandDatasetImage(ctx, onDemandDataset.Accession, imageAccession); err != nil {
				slog.Warn("failed to insert on demand dataset image", "error", err, "origin-accession", originAccession, "image-accession", imageAccession)

				return nil, connect.NewError(connect.CodeInternal, nil)
			}
		}
	}

	if err := on_demand_dataset_metadata_file_handler.RegisterOnDemandDataset(ctx, onDemandDataset); err != nil {
		slog.Warn("failed to register on demand dataset", "error", err)

		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	if err := tx.Commit(); err != nil {
		slog.Warn("failed to commit database transaction", "error", err)

		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	return connect.NewResponse(&dodservice.RequestDatasetCreationResponse{
		OnDemandDatasetAccession: onDemandDataset.Accession,
	}), nil
}

func (d *dodServiceImpl) GetDoDDatasetStatus(ctx context.Context, c *connect.Request[dodservice.GetDoDDatasetStatusRequest]) (*connect.Response[dodservice.GetDoDDatasetStatusResponse], error) {
	ctx, span := observability.Tracer().Start(ctx, "GetDoDDatasetStatus", trace.WithAttributes(attribute.String("accession", c.Msg.GetOnDemandDatasetAccession())))
	defer span.End()

	if c.Msg.GetOnDemandDatasetAccession() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("empty dod dataset accession"))
	}

	dodDatasetReleased, err := database.IsOnDemandDatasetPublished(ctx, c.Msg.GetOnDemandDatasetAccession())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		slog.Warn("failed to check if on demand dataset is publish", "error", err, "accession", c.Msg.GetOnDemandDatasetAccession())

		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	res := dodservice.GetDoDDatasetStatusResponse{
		Status: dodservice.GetDoDDatasetStatusResponse_STATUS_CREATING,
	}

	if dodDatasetReleased {
		res.Status = dodservice.GetDoDDatasetStatusResponse_STATUS_RELEASED
	}

	return connect.NewResponse(&res), nil
}
