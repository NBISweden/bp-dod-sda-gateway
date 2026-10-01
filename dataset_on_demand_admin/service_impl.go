package dataset_on_demand_admin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"github.com/NBISweden/bp-dod-sda-gateway/database"
	dodservice "github.com/NBISweden/bp-dod-sda-gateway/grpc_gen/go/private/v1"
	"github.com/NBISweden/bp-dod-sda-gateway/models"
	"github.com/NBISweden/bp-dod-sda-gateway/models/metadata_models"
	"github.com/NBISweden/bp-dod-sda-gateway/origin_dataset_file_loader"
	"github.com/NBISweden/bp-dod-sda-gateway/pkg/observability"
	"go.opentelemetry.io/otel/attribute"
)

type serviceImpl struct {
	originDatasetFileLoader origin_dataset_file_loader.OriginDatasetFileLoader
}

func (d *serviceImpl) NewOriginDataset(ctx context.Context, c *connect.Request[dodservice.NewOriginDatasetRequest]) (*connect.Response[dodservice.NewOriginDatasetResponse], error) {
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
