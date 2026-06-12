package dod_service_impl

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/imi-bigpicture/bp-dod-sda-gateway/database"
	"github.com/imi-bigpicture/bp-dod-sda-gateway/dod_metadata_file_handler"
	v1 "github.com/imi-bigpicture/bp-dod-sda-gateway/grpc_gen/go/v1"
	"github.com/imi-bigpicture/bp-dod-sda-gateway/grpc_gen/go/v1/v1connect"
	"github.com/imi-bigpicture/bp-dod-sda-gateway/internal/observability"
	"github.com/imi-bigpicture/bp-dod-sda-gateway/models"
	"github.com/imi-bigpicture/bp-dod-sda-gateway/models/metadata_models"
	"github.com/imi-bigpicture/bp-dod-sda-gateway/origin_dataset_file_loader"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type dodServiceImpl struct {
	originDatasetFileLoader origin_dataset_file_loader.OriginDatasetFileLoader
}

func NewDodServiceImpl(options ...func(*dodServiceImpl)) (v1connect.DatasetOnDemandServiceHandler, error) {
	serviceImpl := &dodServiceImpl{}
	for _, o := range options {
		o(serviceImpl)
	}

	if serviceImpl.originDatasetFileLoader == nil {
		return nil, errors.New("originDatasetFileLoader is required")
	}

	return serviceImpl, nil
}

func (d *dodServiceImpl) NewOriginDataset(ctx context.Context, c *connect.Request[v1.NewOriginDatasetRequest]) (*connect.Response[v1.NewOriginDatasetResponse], error) {
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
		Dataset:            new(metadata_models.DatasetSet),
		Image:              new(metadata_models.ImageSet),
		Annotation:         nil, // Optional, populated if present
		Observation:        new(metadata_models.ObservationSet),
		Observer:           nil, // Optional, populated if present
		Policy:             new(metadata_models.PolicySet),
		Sample:             new(metadata_models.SampleSet),
		Staining:           new(metadata_models.StainingSet),
	}

	// Download metadata files, and extract file accession ids
	for _, file := range datasetFiles {
		var err error
		switch file.MetadataFileType {
		case metadata_models.MetadataFileTypeDataset:
			err = d.originDatasetFileLoader.UnmarshalFileToXml(ctx, file, originDataset.Dataset)
		case metadata_models.MetadataFileTypeImage:
			err = d.originDatasetFileLoader.UnmarshalFileToXml(ctx, file, originDataset.Image)
		case metadata_models.MetadataFileTypeAnnotation:
			originDataset.Annotation = new(metadata_models.AnnotationSet)
			err = d.originDatasetFileLoader.UnmarshalFileToXml(ctx, file, originDataset.Annotation)
		case metadata_models.MetadataFileTypeObservation:
			err = d.originDatasetFileLoader.UnmarshalFileToXml(ctx, file, originDataset.Observation)
		case metadata_models.MetadataFileTypeObserver:
			originDataset.Observer = new(metadata_models.ObserverSet)
			err = d.originDatasetFileLoader.UnmarshalFileToXml(ctx, file, originDataset.Observer)
		case metadata_models.MetadataFileTypePolicy:
			err = d.originDatasetFileLoader.UnmarshalFileToXml(ctx, file, originDataset.Policy)
		case metadata_models.MetadataFileTypeSample:
			err = d.originDatasetFileLoader.UnmarshalFileToXml(ctx, file, originDataset.Sample)
		case metadata_models.MetadataFileTypeStaining:
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
		if err := tx.InsertDatasetImage(ctx, originDataset.Accession, image.Alias); err != nil {
			slog.Warn("failed to insert dataset image to database", "error", err, "dataset-accession", originDataset.Accession, "image-alias", image.Alias)

			return nil, connect.NewError(connect.CodeInternal, nil)
		}
		for _, file := range image.Files.Files {
			var found bool
			for downloadPath, fileAccession := range fileAccessions {
				if strings.HasSuffix(strings.TrimSuffix(downloadPath, ".c4gh"), strings.TrimSuffix(file.Filename, ".c4gh")) {
					delete(fileAccessions, downloadPath)
					found = true
					if err := tx.InsertImageFile(ctx, originDataset.Accession, image.Alias, fileAccession); err != nil {
						slog.Warn("failed to insert image file to database", "error", err, "dataset-accession", originDataset.Accession, "image-alias", image.Alias, "file-accession", fileAccession)

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

	return connect.NewResponse(&v1.NewOriginDatasetResponse{}), nil
}

func (d *dodServiceImpl) RequestDatasetCreation(ctx context.Context, c *connect.Request[v1.RequestDatasetCreationRequest]) (*connect.Response[v1.RequestDatasetCreationResponse], error) {
	ctx, span := observability.Tracer().Start(ctx, "RequestDatasetCreation")
	defer span.End()

	if len(c.Msg.GetImageAliases()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("no image aliases requested"))
	}
	if c.Msg.GetUser() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("user requesting dataset on demand must be provided"))
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

	for _, imageAlias := range c.Msg.GetImageAliases() {

		originDatasetAccession, err := tx.GetOriginDatasetAccessionFromImageAlias(ctx, imageAlias)
		if err != nil {
			slog.Warn("failed to get origin dataset accession from image alias", "error", err, "image-alias", imageAlias)

			return nil, connect.NewError(connect.CodeInternal, nil)
		}

		if originDatasetAccession == "" {
			slog.Info("no dataset found from image alias", "image-alias", imageAlias)

			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		if originDatasetImages[originDatasetAccession] == nil {
			originDatasetImages[originDatasetAccession] = []string{imageAlias}

			continue
		}

		originDatasetImages[originDatasetAccession] = append(originDatasetImages[originDatasetAccession], imageAlias)
	}

	originDatasets := make(map[string]*models.OriginDataset)

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
	}

	dodDataset := buildDoDDataset(ctx, originDatasets, originDatasetImages)
	dodDataset.RequestedByUser = c.Msg.GetUser()

	if err := tx.InsertDatasetOnDemandDataset(ctx, dodDataset); err != nil {
		slog.Warn("failed to insert dataset on demand", "error", err)

		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	for originAccession, imageAliases := range originDatasetImages {
		for _, imageAlias := range imageAliases {
			if err := tx.InsertDatasetOnDemandDatasetImage(ctx, dodDataset.Accession, originAccession, imageAlias); err != nil {
				slog.Warn("failed to insert dataset on demand dataset image", "error", err, "origin-accession", originAccession, "image-alias", imageAlias)

				return nil, connect.NewError(connect.CodeInternal, nil)
			}
		}
	}

	if err := dod_metadata_file_handler.RegisterDatasetOnDemandDataset(ctx, dodDataset); err != nil {
		slog.Warn("failed to register dataset on demand dataset", "error", err)

		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	if err := tx.Commit(); err != nil {
		slog.Warn("failed to commit database transaction", "error", err)

		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	return connect.NewResponse(&v1.RequestDatasetCreationResponse{
		DodDatasetAccession: dodDataset.Accession,
	}), nil
}

func (d *dodServiceImpl) GetDoDDataset(ctx context.Context, c *connect.Request[v1.GetDoDDatasetRequest]) (*connect.Response[v1.GetDoDDatasetResponse], error) {
	//TODO implement me
	panic("implement me")
}

func (d *dodServiceImpl) ListDoDDatasets(ctx context.Context, c *connect.Request[v1.ListDoDDatasetsRequest]) (*connect.Response[v1.ListDoDDatasetsResponse], error) {
	//TODO implement me
	panic("implement me")
}

func buildDoDDataset(ctx context.Context, originDatasets map[string]*models.OriginDataset, datasetImages map[string][]string) *models.DatasetOnDemandDataset {
	ctx, span := observability.Tracer().Start(ctx, "buildDoDDataset")
	defer span.End()

	datasetAccession := uuid.NewString()
	policyAccession := uuid.NewString()

	// TODO accession format
	dodDatasetMetadata := &models.DatasetOnDemandDataset{
		Accession:               datasetAccession,
		OriginDatasetAccessions: make([]string, 0, len(originDatasets)),
		DatasetMetadata: models.MetadataFile[*metadata_models.DatasetSet]{
			Accession:   uuid.NewString(),
			MetadataSet: nil, // Populated based on selected images and any linked observations
		},
		ImageMetadata: models.MetadataFile[*metadata_models.ImageSet]{
			Accession:   uuid.NewString(),
			MetadataSet: new(metadata_models.ImageSet),
		},
		ObservationMetadata: models.MetadataFile[*metadata_models.ObservationSet]{
			Accession:   uuid.NewString(),
			MetadataSet: new(metadata_models.ObservationSet),
		},
		ObserverMetadata: nil, // Populated if observers are to be included in dod dataset
		PolicyMetadata: models.MetadataFile[*metadata_models.PolicySet]{
			Accession: uuid.NewString(),
			MetadataSet: &metadata_models.PolicySet{
				Policies: []metadata_models.Policy{
					{
						ObjectType: metadata_models.ObjectType{
							Alias:     policyAccession,
							Accession: policyAccession,
						},
						DatasetRef: metadata_models.Reference{
							Alias:     datasetAccession,
							Accession: datasetAccession,
						},
						Attributes: metadata_models.NullableAttributes{
							Value: &metadata_models.Attributes{
								// Not set as not set as required in Metadata Standards v2.0.0

								StringAttributes: []metadata_models.StringAttribute{
									{
										Tag: "title",
										Value: &metadata_models.NullableString{
											Value: new("Dataset On Demand Policy"),
										},
									}, {
										Tag: "type_of_dataset",
										Value: &metadata_models.NullableString{
											Value: new("Dataset On Demand"), // TODO verify (add to Metadata standard v3.0.0?
										},
									}, {
										Tag: "type_of_access",
										Value: &metadata_models.NullableString{
											Value: new("Direct access"),
										},
									}, {
										Tag: "policy_text",
										Value: &metadata_models.NullableString{
											Value: new("Dataset On Demand Policy"), // TODO
										},
									}, {
										Tag: "legal_basis_for_sharing_the_data",
										Value: &metadata_models.NullableString{
											// TODO
											Nil: true,
										},
									}, {
										Tag: "terms_of_use_version",
										Value: &metadata_models.NullableString{
											Value: new("2026-06-09"),
										},
									}, {
										Tag: "allowed_geographical_distribution",
										Value: &metadata_models.NullableString{
											Value: new("European Union and EEA countries"),
										},
									}, {
										Tag: "duration_of_use",
										Value: &metadata_models.NullableString{
											Value: new("Limited to the duration of the purpose of the data access request"),
										},
									}, {
										Tag: "defined_research_question_required",
										Value: &metadata_models.NullableString{
											Value: new("False"),
										},
									}, {
										Tag: "informed_consent_form_defined_use_restrictions",
										Value: &metadata_models.NullableString{
											Nil: true,
										},
									}, {
										Tag: "custom_use_restrictions",
										Value: &metadata_models.NullableString{
											Nil: true,
										},
									}, {
										Tag: "required_bigpicture_acknowledgment",
										Value: &metadata_models.NullableString{
											Value: new("This project has received funding from the Innovative Medicines Initiative 2 Joint Undertaking under grant agreement No 945358. This Joint Undertaking receives support from the European Union’s Horizon 2020 research and innovation program and EFPIA"),
										},
									}, {
										Tag: "",
										Value: &metadata_models.NullableString{
											Nil: true,
										},
									}, {
										Tag:   "informed_consent_form_defined_use_restrictions",
										Value: &metadata_models.NullableString{},
									},
								},
								NumericAttributes:     nil,
								MeasurementAttributes: nil,
								CodeAttributes:        nil,
								SetAttributes: []metadata_models.SetAttribute{
									{
										Tag: "allowed_uses",
										Value: &metadata_models.NullableAttributes{
											Value: &metadata_models.Attributes{
												StringAttributes: []metadata_models.StringAttribute{
													{ // TODO verify
														Tag: "allowed_use",
														Value: &metadata_models.NullableString{
															Value: new("Implementation of Bigpicture Project (IMI2-945358)"),
														},
													},
												},
											},
										},
									}, {
										Tag: "required_custom_acknowledgments",
										Value: &metadata_models.NullableAttributes{
											Nil: true,
										},
									}, {
										Tag: "required_citations",
										Value: &metadata_models.NullableAttributes{
											Nil: true,
										},
									}, {
										Tag: "licenses", // TODO description missing in Metadata standard v2.0.0
										Value: &metadata_models.NullableAttributes{
											Nil: true,
										},
									},
								},
							},
						},
					},
				},
			},
		},
		RemsMetadata: models.MetadataFile[*metadata_models.RemsSet]{
			Accession:   uuid.NewString(),
			MetadataSet: new(metadata_models.RemsSet),
		},
		SampleMetadata: models.MetadataFile[*metadata_models.SampleSet]{
			Accession:   uuid.NewString(),
			MetadataSet: new(metadata_models.SampleSet),
		},
		StainingMetadata: models.MetadataFile[*metadata_models.StainingSet]{
			Accession:   uuid.NewString(),
			MetadataSet: new(metadata_models.StainingSet),
		},
	}

	// store objects in map with alias -> object to ensure no duplicates in case multiple images reference same entities
	slides := make(map[string]metadata_models.Slide)
	blocks := make(map[string]metadata_models.Block)
	specimens := make(map[string]metadata_models.Specimen)
	biologicalBeings := make(map[string]metadata_models.BiologicalBeing)
	cases := make(map[string]metadata_models.Case)
	stainings := make(map[string]metadata_models.Staining)
	observations := make(map[string]metadata_models.Observation)
	observers := make(map[string]metadata_models.Observer)
	remsEntries := make(map[int]metadata_models.Rems)

	// TODO alias across datasets, generate, etc
	for originDatasetAccession, originDataset := range originDatasets {
		dodDatasetMetadata.OriginDatasetAccessions = append(dodDatasetMetadata.OriginDatasetAccessions, originDatasetAccession)

		if remsEntry, ok := remsEntries[originDataset.RemsWorkflowID]; ok {
			remsEntry.Attributes.Value.SetAttributes[0].Value.Value.StringAttributes = append(
				remsEntry.Attributes.Value.SetAttributes[0].Value.Value.StringAttributes,
				metadata_models.StringAttribute{
					Tag: "origin_dataset",
					Value: &metadata_models.NullableString{
						Value: &datasetAccession,
					},
				},
			)
		} else {
			remsAccession := uuid.NewString()
			remsEntries[originDataset.RemsWorkflowID] = metadata_models.Rems{
				ObjectType: metadata_models.ObjectType{
					Alias:     remsAccession,
					Accession: remsAccession,
				},
				WorkflowId:     fmt.Sprintf("%d", originDataset.RemsWorkflowID),
				OrganisationId: originDataset.RemsOrganisationID,
				DatasetRef: metadata_models.Reference{
					Alias:     datasetAccession,
					Accession: datasetAccession,
				},
				Attributes: &metadata_models.NullableAttributes{
					Value: &metadata_models.Attributes{
						StringAttributes:      nil,
						NumericAttributes:     nil,
						MeasurementAttributes: nil,
						CodeAttributes:        nil,
						SetAttributes: []metadata_models.SetAttribute{
							{
								Tag: "origin_datasets",
								Value: &metadata_models.NullableAttributes{
									Value: &metadata_models.Attributes{
										StringAttributes: []metadata_models.StringAttribute{
											{
												Tag: "origin_dataset",
												Value: &metadata_models.NullableString{
													Value: &datasetAccession,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}
		}

		imagesInDataset := datasetImages[originDatasetAccession]

		for _, imageAlias := range imagesInDataset {

			var slideAlias string

			for _, originImage := range originDataset.Image.Images {
				if originImage.Alias != imageAlias {
					continue
				}
				slideAlias = originImage.ImageOf.Alias
				dodDatasetMetadata.ImageMetadata.MetadataSet.Images = append(dodDatasetMetadata.ImageMetadata.MetadataSet.Images, originImage)

				break
			}

			var blockAlias string
			var stainingAlias string

			for _, originSlide := range originDataset.Sample.Slides {
				if originSlide.Alias != slideAlias {
					continue
				}
				slides[slideAlias] = originSlide
				blockAlias = originSlide.CreatedFromRef.Alias
				stainingAlias = originSlide.StainingInformationRef.Alias

				break
			}

			var specimenAliases []string

			for _, originBlock := range originDataset.Sample.Blocks {
				if originBlock.Alias != blockAlias {
					continue
				}
				blocks[blockAlias] = originBlock
				for _, specimenRef := range originBlock.SampledFromRef {
					specimenAliases = append(specimenAliases, specimenRef.Alias)
				}

				break
			}

			var biologicalBeingsAlias string
			var caseAlias string

			currentSpecimenCount := len(specimens)
			for _, originSpecimen := range originDataset.Sample.Specimens {
				if currentSpecimenCount+len(specimenAliases) == len(specimens) {
					break
				}
				var found bool
				for _, specimenAlias := range specimenAliases {
					if originSpecimen.Alias == specimenAlias {
						found = true
						break
					}
				}
				if !found {
					continue
				}

				specimens[originSpecimen.Alias] = originSpecimen
				biologicalBeingsAlias = originSpecimen.ExtractedFromRef.Alias
				if originSpecimen.PartOfCaseRef != nil {
					caseAlias = originSpecimen.PartOfCaseRef.Alias
				}
			}

			for _, originBiologicalBeing := range originDataset.Sample.BiologicalBeings {
				if originBiologicalBeing.Alias != biologicalBeingsAlias {
					continue
				}
				biologicalBeings[originBiologicalBeing.Alias] = originBiologicalBeing

				break
			}

			for _, originCase := range originDataset.Sample.Cases {
				if originCase.Alias != caseAlias {
					continue
				}
				cases[originCase.Alias] = originCase

				break
			}

			for _, originStain := range originDataset.Staining.Staining {
				if originStain.Alias != stainingAlias {
					continue
				}
				stainings[originStain.Alias] = originStain

				break
			}

			var observerAliases []string
			for _, originObservation := range originDataset.Observation.Observations {
				if (originObservation.CaseRef != nil && originObservation.CaseRef.Alias == caseAlias) ||
					(originObservation.SlideRef != nil && originObservation.SlideRef.Alias == slideAlias) ||
					(originObservation.BlockRef != nil && originObservation.BlockRef.Alias == blockAlias) ||
					(originObservation.BiologicalBeingRef != nil && originObservation.BiologicalBeingRef.Alias == biologicalBeingsAlias) ||
					(originObservation.ImageRef != nil && originObservation.ImageRef.Alias == imageAlias) {

					observations[originObservation.Alias] = originObservation
					for _, observerReference := range originObservation.ObserverRef {
						observerAliases = append(observerAliases, observerReference.Alias)
					}

					continue
				}

				if originObservation.SpecimenRef != nil {
					for _, specimenAlias := range specimenAliases {
						if originObservation.SpecimenRef.Alias != specimenAlias {
							continue
						}

						observations[originObservation.Alias] = originObservation
						for _, observerReference := range originObservation.ObserverRef {
							observerAliases = append(observerAliases, observerReference.Alias)
						}
						break
					}
				}
			}

			if len(observerAliases) == 0 {
				continue
			}

			for _, originObserver := range originDataset.Observer.Observers {
				for _, observerAlias := range observerAliases {
					if originObserver.Alias != observerAlias {
						continue
					}
					observers[originObserver.Alias] = originObserver

					break
				}
			}

		}
	}

	dodDatasetMetadata.DatasetMetadata.MetadataSet = &metadata_models.DatasetSet{
		Dataset: []metadata_models.Dataset{
			{
				ObjectType: metadata_models.ObjectType{
					Alias:     dodDatasetMetadata.Accession,
					Accession: dodDatasetMetadata.Accession,
				},
				Title:                    "Dataset On Demand",
				ShortName:                "Dataset On Demand",
				Description:              nil,
				Version:                  "1.0.0",
				MetadataStandard:         "2.0.0",
				DatasetOwnerContactEmail: nil,
				DatasetType:              []string{"Whole slide imaging"},
				ImageRef:                 nil,
				AnnotationRef:            nil,
				ObservationRef:           nil,
				ComplementsDatasetRef:    nil,
				Attributes:               nil,
			},
		},
	}

	for _, image := range dodDatasetMetadata.ImageMetadata.MetadataSet.Images {
		dodDatasetMetadata.DatasetMetadata.MetadataSet.Dataset[0].ImageRef = append(dodDatasetMetadata.DatasetMetadata.MetadataSet.Dataset[0].ImageRef, metadata_models.Reference{
			Alias:     image.Alias,
			Accession: image.Accession,
		})
	}

	for _, slide := range slides {
		dodDatasetMetadata.SampleMetadata.MetadataSet.Slides = append(dodDatasetMetadata.SampleMetadata.MetadataSet.Slides, slide)
	}
	for _, block := range blocks {
		dodDatasetMetadata.SampleMetadata.MetadataSet.Blocks = append(dodDatasetMetadata.SampleMetadata.MetadataSet.Blocks, block)
	}
	for _, specimen := range specimens {
		dodDatasetMetadata.SampleMetadata.MetadataSet.Specimens = append(dodDatasetMetadata.SampleMetadata.MetadataSet.Specimens, specimen)
	}
	for _, biologicalBeing := range biologicalBeings {
		dodDatasetMetadata.SampleMetadata.MetadataSet.BiologicalBeings = append(dodDatasetMetadata.SampleMetadata.MetadataSet.BiologicalBeings, biologicalBeing)
	}
	for _, mdCase := range cases {
		dodDatasetMetadata.SampleMetadata.MetadataSet.Cases = append(dodDatasetMetadata.SampleMetadata.MetadataSet.Cases, mdCase)
	}
	for _, staining := range stainings {
		dodDatasetMetadata.StainingMetadata.MetadataSet.Staining = append(dodDatasetMetadata.StainingMetadata.MetadataSet.Staining, staining)
	}

	for _, observation := range observations {
		dodDatasetMetadata.ObservationMetadata.MetadataSet.Observations = append(dodDatasetMetadata.ObservationMetadata.MetadataSet.Observations, observation)

		dodDatasetMetadata.DatasetMetadata.MetadataSet.Dataset[0].ObservationRef = append(dodDatasetMetadata.DatasetMetadata.MetadataSet.Dataset[0].ObservationRef, metadata_models.Reference{
			Alias:     observation.Alias,
			Accession: observation.Accession,
		})
	}

	if len(observers) > 0 {
		dodDatasetMetadata.ObserverMetadata = &models.MetadataFile[*metadata_models.ObserverSet]{
			Accession: uuid.NewString(),
			MetadataSet: &metadata_models.ObserverSet{
				Observers: make([]metadata_models.Observer, len(observers)),
			},
		}
	}
	for _, observer := range observers {
		dodDatasetMetadata.ObserverMetadata.MetadataSet.Observers = append(dodDatasetMetadata.ObserverMetadata.MetadataSet.Observers, observer)
	}

	for _, rems := range remsEntries {
		dodDatasetMetadata.RemsMetadata.MetadataSet.Rems = append(dodDatasetMetadata.RemsMetadata.MetadataSet.Rems, rems)
	}

	return dodDatasetMetadata
}
