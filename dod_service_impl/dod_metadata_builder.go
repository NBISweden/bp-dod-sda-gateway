package dod_service_impl

import (
	"context"
	"fmt"
	"strings"

	"github.com/NBISweden/bp-dod-sda-gateway/models"
	"github.com/NBISweden/bp-dod-sda-gateway/models/metadata_models"
	"github.com/NBISweden/bp-dod-sda-gateway/pkg/accession_generation"
	"github.com/NBISweden/bp-dod-sda-gateway/pkg/observability"
)

// buildDoDDataset builds the Dataset On Demand Dataset from the origin dataset and the selected images
// All the aliases in the resulting metadata will be replaced with the relative accession to ensure the aliases for the entities are unique within the created dataset
func buildDoDDataset(ctx context.Context, originDatasets map[string]*models.OriginDataset, datasetImages map[string][]string) *models.DatasetOnDemandDataset {
	_, span := observability.Tracer().Start(ctx, "buildDoDDataset")
	defer span.End()

	datasetAccession := accession_generation.GenerateAccessionID(accession_generation.Dataset)

	dodDatasetMetadata := &models.DatasetOnDemandDataset{
		Accession:               datasetAccession,
		OriginDatasetAccessions: make([]string, 0, len(originDatasets)),
		DatasetMetadata: models.MetadataFile[*metadata_models.DatasetSet]{
			Accession:   accession_generation.GenerateAccessionID(accession_generation.File),
			MetadataSet: nil, // Populated based on selected images and any linked observations
		},
		ImageMetadata: models.MetadataFile[*metadata_models.ImageSet]{
			Accession:   accession_generation.GenerateAccessionID(accession_generation.File),
			MetadataSet: new(metadata_models.ImageSet),
		},
		ObservationMetadata: models.MetadataFile[*metadata_models.ObservationSet]{
			Accession:   accession_generation.GenerateAccessionID(accession_generation.File),
			MetadataSet: new(metadata_models.ObservationSet),
		},
		ObserverMetadata: nil, // Populated if observers are to be included in dod dataset
		PolicyMetadata: models.MetadataFile[*metadata_models.PolicySet]{
			Accession:   accession_generation.GenerateAccessionID(accession_generation.File),
			MetadataSet: &metadata_models.PolicySet{},
		},
		RemsMetadata: models.MetadataFile[*metadata_models.RemsSet]{
			Accession:   accession_generation.GenerateAccessionID(accession_generation.File),
			MetadataSet: new(metadata_models.RemsSet),
		},
		SampleMetadata: models.MetadataFile[*metadata_models.SampleSet]{
			Accession:   accession_generation.GenerateAccessionID(accession_generation.File),
			MetadataSet: new(metadata_models.SampleSet),
		},
		StainingMetadata: models.MetadataFile[*metadata_models.StainingSet]{
			Accession:   accession_generation.GenerateAccessionID(accession_generation.File),
			MetadataSet: new(metadata_models.StainingSet),
		},
	}

	// store objects in map with accession -> object to ensure no duplicates in case multiple images reference same entities
	slides := make(map[string]metadata_models.Slide)
	blocks := make(map[string]metadata_models.Block)
	specimens := make(map[string]metadata_models.Specimen)
	biologicalBeings := make(map[string]metadata_models.BiologicalBeing)
	cases := make(map[string]metadata_models.Case)
	stainings := make(map[string]metadata_models.Staining)
	observations := make(map[string]metadata_models.Observation)
	observers := make(map[string]metadata_models.Observer)
	remsEntries := make(map[int]metadata_models.Rems)

	for originDatasetAccession, originDataset := range originDatasets {
		dodDatasetMetadata.OriginDatasetAccessions = append(dodDatasetMetadata.OriginDatasetAccessions, originDatasetAccession)

		// Just add policies from one rigin dataset as they should all be the same
		if len(originDataset.Policy.Policies) != 0 {
			dodDatasetMetadata.PolicyMetadata.MetadataSet.Policies = originDataset.Policy.Policies
			dodDatasetMetadata.PolicyMetadata.MetadataSet.Policies[0].DatasetRef = metadata_models.Reference{
				Alias:     dodDatasetMetadata.Accession,
				Accession: dodDatasetMetadata.Accession,
			}
		}

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
			remsAccession := accession_generation.GenerateAccessionID(accession_generation.File)
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

		for _, imageAccession := range imagesInDataset {

			var slideAccession string

			for _, originImage := range originDataset.Image.Images {
				if originImage.Accession != imageAccession {
					continue
				}
				slideAccession = originImage.ImageOf.Accession

				for i, imageFile := range originImage.Files.Files {
					originImage.Files.Files[i].Filename = strings.ReplaceAll(imageFile.Filename, originImage.Alias, originImage.Accession)
				}
				originImage.Alias = originImage.Accession
				originImage.ImageOf.Alias = originImage.ImageOf.Accession
				dodDatasetMetadata.ImageMetadata.MetadataSet.Images = append(dodDatasetMetadata.ImageMetadata.MetadataSet.Images, originImage)

				break
			}

			var blockAccession string
			var stainingAccession string

			for _, originSlide := range originDataset.Sample.Slides {
				if originSlide.Accession != slideAccession {
					continue
				}
				blockAccession = originSlide.CreatedFromRef.Accession
				stainingAccession = originSlide.StainingInformationRef.Accession

				originSlide.Alias = originSlide.Accession
				originSlide.CreatedFromRef.Alias = originSlide.CreatedFromRef.Accession
				originSlide.StainingInformationRef.Alias = originSlide.StainingInformationRef.Accession
				slides[slideAccession] = originSlide

				break
			}

			var specimenAccessions []string

			for _, originBlock := range originDataset.Sample.Blocks {
				if originBlock.Accession != blockAccession {
					continue
				}
				originBlock.Alias = originBlock.Accession
				for i, specimenRef := range originBlock.SampledFromRef {
					originBlock.SampledFromRef[i].Alias = originBlock.SampledFromRef[i].Accession
					specimenAccessions = append(specimenAccessions, specimenRef.Accession)
				}
				blocks[blockAccession] = originBlock

				break
			}

			var biologicalBeingsAccession string
			var caseAccession string

			currentSpecimenCount := len(specimens)
			for _, originSpecimen := range originDataset.Sample.Specimens {
				if currentSpecimenCount+len(specimenAccessions) == len(specimens) {
					break
				}
				var found bool
				for _, specimenAccession := range specimenAccessions {
					if originSpecimen.Accession == specimenAccession {
						found = true
						break
					}
				}
				if !found {
					continue
				}
				biologicalBeingsAccession = originSpecimen.ExtractedFromRef.Accession
				if originSpecimen.PartOfCaseRef != nil {
					originSpecimen.PartOfCaseRef.Alias = originSpecimen.PartOfCaseRef.Accession
					caseAccession = originSpecimen.PartOfCaseRef.Accession
				}
				originSpecimen.Alias = originSpecimen.Accession
				originSpecimen.ExtractedFromRef.Alias = originSpecimen.ExtractedFromRef.Accession
				specimens[originSpecimen.Accession] = originSpecimen
			}

			for _, originCase := range originDataset.Sample.Cases {
				if originCase.Accession != caseAccession {
					continue
				}
				originCase.Alias = originCase.Accession
				originCase.BiologicalBeingRef.Alias = originCase.BiologicalBeingRef.Accession
				cases[originCase.Accession] = originCase

				break
			}

			for _, originBiologicalBeing := range originDataset.Sample.BiologicalBeings {
				if originBiologicalBeing.Accession != biologicalBeingsAccession {
					continue
				}
				originBiologicalBeing.Alias = originBiologicalBeing.Accession
				biologicalBeings[originBiologicalBeing.Accession] = originBiologicalBeing

				break
			}

			for _, originStain := range originDataset.Staining.Staining {
				if originStain.Accession != stainingAccession {
					continue
				}
				originStain.Alias = originStain.Accession
				stainings[originStain.Accession] = originStain

				break
			}

			var observerAccessions []string
			for _, originObservation := range originDataset.Observation.Observations {
				if (originObservation.CaseRef != nil && originObservation.CaseRef.Accession == caseAccession) ||
					(originObservation.SlideRef != nil && originObservation.SlideRef.Accession == slideAccession) ||
					(originObservation.BlockRef != nil && originObservation.BlockRef.Accession == blockAccession) ||
					(originObservation.BiologicalBeingRef != nil && originObservation.BiologicalBeingRef.Accession == biologicalBeingsAccession) ||
					(originObservation.ImageRef != nil && originObservation.ImageRef.Accession == imageAccession) {

					for i, observerReference := range originObservation.ObserverRef {
						originObservation.ObserverRef[i].Alias = originObservation.ObserverRef[i].Accession
						observerAccessions = append(observerAccessions, observerReference.Accession)
					}
					originObservation.Alias = originObservation.Accession
					if originObservation.CaseRef != nil {
						originObservation.CaseRef.Alias = originObservation.CaseRef.Accession
					}
					if originObservation.SlideRef != nil {
						originObservation.SlideRef.Alias = originObservation.SlideRef.Accession
					}
					if originObservation.BlockRef != nil {
						originObservation.BlockRef.Alias = originObservation.BlockRef.Accession
					}
					if originObservation.BiologicalBeingRef != nil {
						originObservation.BiologicalBeingRef.Alias = originObservation.BiologicalBeingRef.Accession
					}
					if originObservation.ImageRef != nil {
						originObservation.ImageRef.Alias = originObservation.ImageRef.Accession
					}
					observations[originObservation.Accession] = originObservation

					continue
				}

				if originObservation.SpecimenRef != nil {
					for _, specimenAccession := range specimenAccessions {
						if originObservation.SpecimenRef.Accession != specimenAccession {
							continue
						}
						for _, observerReference := range originObservation.ObserverRef {
							observerAccessions = append(observerAccessions, observerReference.Accession)
						}
						originObservation.Alias = originObservation.Accession
						originObservation.SpecimenRef.Alias = originObservation.SpecimenRef.Accession
						observations[originObservation.Accession] = originObservation

						break
					}
				}
			}

			if len(observerAccessions) == 0 {
				continue
			}

			for _, originObserver := range originDataset.Observer.Observers {
				for _, observerAccession := range observerAccessions {
					if originObserver.Accession != observerAccession {
						continue
					}
					originObserver.Alias = originObserver.Accession
					observers[originObserver.Accession] = originObserver

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
			Accession: accession_generation.GenerateAccessionID(accession_generation.File),
			MetadataSet: &metadata_models.ObserverSet{
				Observers: make([]metadata_models.Observer, 0, len(observers)),
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
