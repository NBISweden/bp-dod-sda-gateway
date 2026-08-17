package dataset_on_demand_service_impl

import (
	"context"
	"testing"

	"github.com/NBISweden/bp-dod-sda-gateway/models"
	"github.com/NBISweden/bp-dod-sda-gateway/models/metadata_models"
	"github.com/stretchr/testify/assert"
)

func TestBuildOnDemandDataset(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		originDatasets       map[string]*models.OriginDataset
		imageAccessions      map[string]map[string]struct{}
		wantImageCount       int
		wantObservationCount int
	}{
		{
			name: "2_images_different_origin_datasets",
			originDatasets: map[string]*models.OriginDataset{
				"dataset1": generateTestOriginDataset("dataset1"),
				"dataset2": generateTestOriginDataset("dataset2"),
				"dataset3": generateTestOriginDataset("dataset3"),
			},
			imageAccessions: map[string]map[string]struct{}{
				"dataset1": {
					"dataset1_image_1_accession": {},
				}, "dataset2": {
					"dataset2_image_1_accession": {},
				},
			},
		}, {
			name: "3_images_different_origin_datasets",
			originDatasets: map[string]*models.OriginDataset{
				"dataset1": generateTestOriginDataset("dataset1"),
				"dataset2": generateTestOriginDataset("dataset2"),
				"dataset3": generateTestOriginDataset("dataset3"),
			},
			imageAccessions: map[string]map[string]struct{}{
				"dataset1": {
					"dataset1_image_1_accession": {},
				}, "dataset2": {
					"dataset2_image_2_accession": {},
				}, "dataset3": {
					"dataset3_image_3_accession": {},
				},
			},
		}, {
			name: "3_images_same_origin_dataset",
			originDatasets: map[string]*models.OriginDataset{
				"dataset1": generateTestOriginDataset("dataset1"),
				"dataset2": generateTestOriginDataset("dataset2"),
				"dataset3": generateTestOriginDataset("dataset3"),
			},
			imageAccessions: map[string]map[string]struct{}{
				"dataset1": {
					"dataset1_image_1_accession": {},
					"dataset1_image_2_accession": {},
					"dataset1_image_3_accession": {},
				},
			},
		}, {
			name: "1_image",
			originDatasets: map[string]*models.OriginDataset{
				"dataset1": generateTestOriginDataset("dataset1"),
				"dataset2": generateTestOriginDataset("dataset2"),
				"dataset3": generateTestOriginDataset("dataset3"),
			},
			imageAccessions: map[string]map[string]struct{}{
				"dataset1": {
					"dataset1_image_1_accession": {},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			onDemandDataset := buildOnDemandDataset(context.Background(), tc.originDatasets, tc.imageAccessions)

			if len(onDemandDataset.DatasetMetadata.MetadataSet.Dataset) != 1 {
				t.Error("OnDemandDataset dataset metadata not 1")
				t.FailNow()
			}
			var expectedImageCount int
			for _, imageAccessions := range tc.imageAccessions {
				for imageAccession := range imageAccessions {
					expectedImageCount++
					var found bool

					for _, imageRef := range onDemandDataset.DatasetMetadata.MetadataSet.Dataset[0].ImageRef {
						if imageAccession != imageRef.Accession {
							continue
						}
						found = true

						assert.Equal(t, imageRef.Alias, imageRef.Accession)

						var slideAccession string

						for _, image := range onDemandDataset.ImageMetadata.MetadataSet.Images {
							if imageRef.Accession != image.Accession {
								continue
							}
							assert.Equal(t, image.Alias, image.Accession)
							assert.Equal(t, image.ImageOf.Alias, image.ImageOf.Accession)
							slideAccession = image.ImageOf.Accession

							break
						}
						if slideAccession == "" {
							t.Errorf("OnDemandDataset image metadata does not contain an expected slide reference for image with accession: %s", imageAccession)
							t.Fail()

							break
						}

						var blockAccession string
						var stainingAccession string

						for _, slide := range onDemandDataset.SampleMetadata.MetadataSet.Slides {
							if slide.Accession != slideAccession {
								continue
							}
							assert.Equal(t, slide.Alias, slide.Accession)
							assert.Equal(t, slide.CreatedFromRef.Alias, slide.CreatedFromRef.Accession)
							assert.Equal(t, slide.StainingInformationRef.Alias, slide.StainingInformationRef.Accession)

							blockAccession = slide.CreatedFromRef.Accession
							stainingAccession = slide.StainingInformationRef.Accession

							break
						}

						if stainingAccession == "" {
							t.Errorf("OnDemandDataset sample metadata does not contain an expected stainging for slide with accession: %s", slideAccession)
							t.Fail()

							break
						}

						if blockAccession == "" {
							t.Errorf("OnDemandDataset sample metadata does not contain an expected block for slide with accession: %s", slideAccession)
							t.Fail()

							break
						}

						var stainingFound bool
						for _, staining := range onDemandDataset.StainingMetadata.MetadataSet.Staining {
							if staining.Accession != stainingAccession {
								continue
							}
							assert.Equal(t, staining.Alias, staining.Accession)
							stainingFound = true

							break
						}
						if !stainingFound {
							t.Errorf("OnDemandDataset staining metadata does not contain an expected staining with accession: %s", stainingAccession)
							t.Fail()

							break
						}

						var specimenRefs []string
						for _, block := range onDemandDataset.SampleMetadata.MetadataSet.Blocks {
							if block.Accession != blockAccession {
								continue
							}
							assert.Equal(t, block.Alias, block.Accession)
							for _, sampledRef := range block.SampledFromRef {
								assert.Equal(t, sampledRef.Alias, sampledRef.Accession)
								specimenRefs = append(specimenRefs, sampledRef.Accession)
							}

							break
						}

						if len(specimenRefs) == 0 {
							t.Errorf("OnDemandDataset sample metadata does not contain any expected specimen refs with block accession: %s", blockAccession)
							t.Fail()

							break
						}

						var biologicalBeingRefs []string
						var caseRefs []string
						for _, specimenRef := range specimenRefs {
							var specimenFound bool

							for _, specimen := range onDemandDataset.SampleMetadata.MetadataSet.Specimens {
								if specimenRef != specimen.Accession {
									continue
								}
								specimenFound = true
								assert.Equal(t, specimen.Alias, specimen.Accession)
								assert.Equal(t, specimen.ExtractedFromRef.Alias, specimen.ExtractedFromRef.Accession)
								biologicalBeingRefs = append(biologicalBeingRefs, specimen.ExtractedFromRef.Accession)

								if specimen.PartOfCaseRef != nil {
									assert.Equal(t, specimen.PartOfCaseRef.Alias, specimen.PartOfCaseRef.Accession)
									caseRefs = append(caseRefs, specimen.PartOfCaseRef.Accession)
								}

								break
							}

							if !specimenFound {
								t.Errorf("OnDemandDataset sample metadata does not contain expected specimen with accession: %s", specimenRef)
								t.Fail()
							}
						}

						for _, biologicalBeingRef := range biologicalBeingRefs {
							var biologicalBeingFound bool

							for _, biologicalBeing := range onDemandDataset.SampleMetadata.MetadataSet.BiologicalBeings {
								if biologicalBeingRef != biologicalBeing.Accession {
									continue
								}
								biologicalBeingFound = true
								assert.Equal(t, biologicalBeing.Alias, biologicalBeing.Accession)

								break
							}

							if !biologicalBeingFound {
								t.Errorf("OnDemandDataset sample metadata does not contain expected biological being with accession: %s", biologicalBeingRef)
								t.Fail()
							}
						}

						for _, caseRef := range caseRefs {
							var caseFound bool

							for _, caseEntity := range onDemandDataset.SampleMetadata.MetadataSet.Cases {
								if caseRef != caseEntity.Accession {
									continue
								}
								caseFound = true
								assert.Equal(t, caseEntity.Alias, caseEntity.Accession)

								break
							}

							if !caseFound {
								t.Errorf("OnDemandDataset sample metadata does not contain expected case with accession: %s", caseRef)
								t.Fail()
							}
						}

						break
					}

					if !found {
						t.Errorf("OnDemandDataset dataset metadata does not contain expected image accession ref: %s", imageAccession)
						t.Fail()
					}
				}
			}

			assert.Equal(t, len(onDemandDataset.DatasetMetadata.MetadataSet.Dataset[0].ImageRef), expectedImageCount)
			assert.Equal(t, len(onDemandDataset.ImageMetadata.MetadataSet.Images), expectedImageCount)
		})
	}
}

func generateTestOriginDataset(datasetName string) *models.OriginDataset {
	attributes := metadata_models.NullableAttributes{
		Value: &metadata_models.Attributes{
			StringAttributes: []metadata_models.StringAttribute{
				{
					Tag: "string_attr_1",
					Value: &metadata_models.NullableString{
						Value: new("value_1"),
						Nil:   false,
					},
				}, {
					Tag: "string_attr_2",
					Value: &metadata_models.NullableString{
						Value: new("value_2"),
						Nil:   false,
					},
				}, {
					Tag: "string_attr_3",
					Value: &metadata_models.NullableString{
						Value: nil,
						Nil:   true,
					},
				},
			},
			NumericAttributes: []metadata_models.NumericAttribute{
				{
					Tag: "numeric_attr_1",
					Value: &metadata_models.NullableFloat{
						Value: new(1.1),
						Nil:   false,
					},
				}, {
					Tag: "numeric_attr_2",
					Value: &metadata_models.NullableFloat{
						Value: nil,
						Nil:   true,
					},
				},
			},
			MeasurementAttributes: []metadata_models.MeasurementAttribute{
				{
					Tag: "measurement_attr_1",
					Value: &metadata_models.NullableFloat{
						Value: new(2.2),
						Nil:   false,
					},
					Units: "test_unit",
				},
			},
			CodeAttributes: []metadata_models.CodeAttribute{
				{
					Tag: "code_attr_1",
					Value: &metadata_models.NullableCodeAttributeValue{
						Value: &metadata_models.CodeAttributeValue{
							Code:    "123",
							Scheme:  "test",
							Meaning: "unit_test",
							SchemeVersion: &metadata_models.NullableString{
								Value: new("unit_test"),
								Nil:   false,
							},
						},
						Nil: false,
					},
				},
			},
			SetAttributes: []metadata_models.SetAttribute{
				{
					Tag: "set_attr_1",
					Value: &metadata_models.NullableAttributes{
						Value: &metadata_models.Attributes{
							StringAttributes: []metadata_models.StringAttribute{
								{
									Tag:   "set_attr_1/string_attr_1",
									Value: nil,
								},
							},
						},
						Nil: false,
					},
				},
			},
		},
		Nil: false,
	}

	return &models.OriginDataset{
		Accession:          "dataset1",
		RemsWorkflowID:     2,
		RemsOrganisationID: "test-org",
		Dataset: &metadata_models.DatasetSet{
			Dataset: []metadata_models.Dataset{
				{
					ObjectType: metadata_models.ObjectType{
						Alias:     datasetName + "_alias",
						Accession: datasetName + "_accession",
					},
					Title:                    datasetName,
					ShortName:                datasetName,
					Description:              nil,
					Version:                  "1.0.0",
					MetadataStandard:         "2.0.0",
					DatasetOwnerContactEmail: nil,
					DatasetType:              nil,
					ImageRef: []metadata_models.Reference{
						{
							Alias:     datasetName + "_image_1_alias",
							Accession: datasetName + "_image_1_accession",
						}, {
							Alias:     datasetName + "_image_2_alias",
							Accession: datasetName + "_image_2_accession",
						}, {
							Alias:     datasetName + "_image_3_alias",
							Accession: datasetName + "_image_3_accession",
						},
					},
					ObservationRef: []metadata_models.Reference{
						{
							Alias:     datasetName + "_observation_1_alias",
							Accession: datasetName + "_observation_1_accession",
						}, {
							Alias:     datasetName + "_observation_2_alias",
							Accession: datasetName + "_observation_2_accession",
						}, {
							Alias:     datasetName + "_observation_3_alias",
							Accession: datasetName + "_observation_3_accession",
						}, {
							Alias:     datasetName + "_observation_4_alias",
							Accession: datasetName + "_observation_4_accession",
						}, {
							Alias:     datasetName + "_observation_5_alias",
							Accession: datasetName + "_observation_5_accession",
						},
					},
					ComplementsDatasetRef: nil,
					Attributes:            &attributes,
				},
			},
		},
		Image: &metadata_models.ImageSet{
			Images: []metadata_models.Image{
				{
					ObjectType: metadata_models.ObjectType{
						Alias:     datasetName + "_image_1_alias",
						Accession: datasetName + "_image_1_accession",
					},
					ImageOf: metadata_models.Reference{
						Alias:     datasetName + "_slide_1_alias",
						Accession: datasetName + "_slide_1_accession",
					},
					ImageType: metadata_models.ImageTypeChoice{},
					Files: metadata_models.ImageFiles{
						Files: []metadata_models.ImageFile{
							{
								FileBaseType: metadata_models.FileBaseType{
									Filename:            datasetName + "/image_1/file_1",
									ChecksumMethod:      "sha256",
									Checksum:            "123",
									UnencryptedChecksum: "123",
								},
								FileType: "dcm",
							}, {
								FileBaseType: metadata_models.FileBaseType{
									Filename:            datasetName + "/image_1/file_2",
									ChecksumMethod:      "sha256",
									Checksum:            "123",
									UnencryptedChecksum: "123",
								},
								FileType: "dcm",
							},
						},
					},
					Attributes: attributes,
				}, {
					ObjectType: metadata_models.ObjectType{
						Alias:     datasetName + "_image_2_alias",
						Accession: datasetName + "_image_2_accession",
					},
					ImageOf: metadata_models.Reference{
						Alias:     datasetName + "_slide_2_alias",
						Accession: datasetName + "_slide_2_accession",
					},
					ImageType: metadata_models.ImageTypeChoice{},
					Files: metadata_models.ImageFiles{
						Files: []metadata_models.ImageFile{
							{
								FileBaseType: metadata_models.FileBaseType{
									Filename:            datasetName + "/image_2/file_1",
									ChecksumMethod:      "sha256",
									Checksum:            "123",
									UnencryptedChecksum: "123",
								},
								FileType: "dcm",
							}, {
								FileBaseType: metadata_models.FileBaseType{
									Filename:            datasetName + "/image_2/file_2",
									ChecksumMethod:      "sha256",
									Checksum:            "123",
									UnencryptedChecksum: "123",
								},
								FileType: "dcm",
							},
						},
					},
					Attributes: attributes,
				}, {
					ObjectType: metadata_models.ObjectType{
						Alias:     datasetName + "_image_3_alias",
						Accession: datasetName + "_image_3_accession",
					},
					ImageOf: metadata_models.Reference{
						Alias:     datasetName + "_slide_2_alias",
						Accession: datasetName + "_slide_2_accession",
					},
					ImageType: metadata_models.ImageTypeChoice{},
					Files: metadata_models.ImageFiles{
						Files: []metadata_models.ImageFile{
							{
								FileBaseType: metadata_models.FileBaseType{
									Filename:            datasetName + "/image_3/file_1",
									ChecksumMethod:      "sha256",
									Checksum:            "123",
									UnencryptedChecksum: "123",
								},
								FileType: "dcm",
							}, {
								FileBaseType: metadata_models.FileBaseType{
									Filename:            datasetName + "/image_3/file_2",
									ChecksumMethod:      "sha256",
									Checksum:            "123",
									UnencryptedChecksum: "123",
								},
								FileType: "dcm",
							},
						},
					},
					Attributes: attributes,
				},
			},
		},
		Annotation: nil,
		Observation: &metadata_models.ObservationSet{
			Observations: []metadata_models.Observation{
				{
					ObjectType: metadata_models.ObjectType{
						Alias:     datasetName + "_observation_1_alias",
						Accession: datasetName + "_observation_1_accession",
					},
					ImageRef: &metadata_models.Reference{
						Alias:     datasetName + "_image_1_alias",
						Accession: datasetName + "_image_1_accession",
					},
					ObserverRef: []metadata_models.Reference{
						{
							Alias:     datasetName + "_observer_1_alias",
							Accession: datasetName + "_observer_1_accession",
						},
					},
					Statement: metadata_models.Statement{
						StatementType:    "test",
						StatementStatus:  "pending",
						CodeAttributes:   nil,
						CustomAttributes: nil,
						Freetext: &metadata_models.NullableString{
							Value: new("test observation"),
							Nil:   false,
						},
						Attributes: nil,
					},
					Attributes: &attributes,
				}, {
					ObjectType: metadata_models.ObjectType{
						Alias:     datasetName + "_observation_2_alias",
						Accession: datasetName + "_observation_2_accession",
					},
					BlockRef: &metadata_models.Reference{
						Alias:     datasetName + "_block_1_alias",
						Accession: datasetName + "_block_1_accession",
					},
					ObserverRef: []metadata_models.Reference{
						{
							Alias:     datasetName + "_observer_1_alias",
							Accession: datasetName + "_observer_1_accession",
						},
					},
					Statement: metadata_models.Statement{
						StatementType:    "test",
						StatementStatus:  "pending",
						CodeAttributes:   nil,
						CustomAttributes: nil,
						Freetext: &metadata_models.NullableString{
							Value: new("test observer observation"),
							Nil:   false,
						},
						Attributes: nil,
					},
					Attributes: nil,
				}, {
					ObjectType: metadata_models.ObjectType{
						Alias:     datasetName + "_observation_3_alias",
						Accession: datasetName + "_observation_3_accession",
					},
					SpecimenRef: &metadata_models.Reference{
						Alias:     datasetName + "_specimen_1_alias",
						Accession: datasetName + "_specimen_1_accession",
					},
					ObserverRef: []metadata_models.Reference{
						{
							Alias:     datasetName + "_observer_1_alias",
							Accession: datasetName + "_observer_1_accession",
						},
					},
					Statement: metadata_models.Statement{
						StatementType:    "test",
						StatementStatus:  "pending",
						CodeAttributes:   nil,
						CustomAttributes: nil,
						Freetext: &metadata_models.NullableString{
							Value: new("test specimen observation"),
							Nil:   false,
						},
						Attributes: nil,
					},
					Attributes: &attributes,
				}, {
					ObjectType: metadata_models.ObjectType{
						Alias:     datasetName + "_observation_4_alias",
						Accession: datasetName + "_observation_4_accession",
					},
					SpecimenRef: &metadata_models.Reference{
						Alias:     datasetName + "_slide_2_alias",
						Accession: datasetName + "_slide_2_accession",
					},
					ObserverRef: []metadata_models.Reference{
						{
							Alias:     datasetName + "_observer_1_alias",
							Accession: datasetName + "_observer_1_accession",
						},
					},
					Statement: metadata_models.Statement{
						StatementType:    "test",
						StatementStatus:  "pending",
						CodeAttributes:   nil,
						CustomAttributes: nil,
						Freetext: &metadata_models.NullableString{
							Value: new("test specimen observation"),
							Nil:   false,
						},
						Attributes: nil,
					},
					Attributes: &attributes,
				}, {
					ObjectType: metadata_models.ObjectType{
						Alias:     datasetName + "_observation_5_alias",
						Accession: datasetName + "_observation_5_accession",
					},
					ImageRef: &metadata_models.Reference{
						Alias:     datasetName + "_specimen_3_alias",
						Accession: datasetName + "_specimen_3_accession",
					},
					ObserverRef: []metadata_models.Reference{
						{
							Alias:     datasetName + "_observer_2_alias",
							Accession: datasetName + "_observer_2_accession",
						},
					},
					Statement: metadata_models.Statement{
						StatementType:    "test",
						StatementStatus:  "pending",
						CodeAttributes:   nil,
						CustomAttributes: nil,
						Freetext: &metadata_models.NullableString{
							Value: new("test image observation"),
							Nil:   false,
						},
						Attributes: nil,
					},
					Attributes: &attributes,
				},
			},
		},
		Observer: &metadata_models.ObserverSet{
			Observers: []metadata_models.Observer{
				{
					ObjectType: metadata_models.ObjectType{
						Alias:     datasetName + "_observer_1_alias",
						Accession: datasetName + "_observer_1_accession",
					},
					ObserverType: "test",
					Attributes:   &attributes,
				}, {
					ObjectType: metadata_models.ObjectType{
						Alias:     datasetName + "_observer_2_alias",
						Accession: datasetName + "_observer_2_accession",
					},
					ObserverType: "test",
					Attributes:   &attributes,
				},
			},
		},
		Policy: &metadata_models.PolicySet{
			Policies: []metadata_models.Policy{
				{
					ObjectType: metadata_models.ObjectType{
						Alias:     datasetName + "_policy_1_alias",
						Accession: datasetName + "_policy_1_accession",
					},
					DatasetRef: metadata_models.Reference{
						Alias:     datasetName + "_alias",
						Accession: datasetName + "_accession",
					},
					Attributes: attributes,
				},
			},
		},
		Sample: &metadata_models.SampleSet{
			BiologicalBeings: []metadata_models.BiologicalBeing{
				{
					ObjectType: metadata_models.ObjectType{
						Alias:     datasetName + "_biological_being_1_alias",
						Accession: datasetName + "_biological_being_1_accession",
					},
					Attributes: attributes,
				}, {
					ObjectType: metadata_models.ObjectType{
						Alias:     datasetName + "_biological_being_2_alias",
						Accession: datasetName + "_biological_being_2_accession",
					},
					Attributes: attributes,
				},
			},
			Cases: []metadata_models.Case{
				{
					ObjectType: metadata_models.ObjectType{
						Alias:     datasetName + "_case_1_alias",
						Accession: datasetName + "_case_1_accession",
					},
					BiologicalBeingRef: metadata_models.Reference{
						Alias:     datasetName + "_biological_being_1_alias",
						Accession: datasetName + "_biological_being_1_accession",
					},
					Attributes: &attributes,
				},
			},
			Specimens: []metadata_models.Specimen{
				{
					ObjectType: metadata_models.ObjectType{
						Alias:     datasetName + "_specimen_1_alias",
						Accession: datasetName + "_specimen_1_accession",
					},
					ExtractedFromRef: metadata_models.Reference{
						Alias:     datasetName + "_biological_being_1_alias",
						Accession: datasetName + "_biological_being_1_accession",
					},
					PartOfCaseRef: &metadata_models.Reference{
						Alias:     datasetName + "_case_1_alias",
						Accession: datasetName + "_case_1_accession",
					},
					Attributes: &attributes,
				}, {
					ObjectType: metadata_models.ObjectType{
						Alias:     datasetName + "_specimen_2_alias",
						Accession: datasetName + "_specimen_2_accession",
					},
					ExtractedFromRef: metadata_models.Reference{
						Alias:     datasetName + "_biological_being_2_alias",
						Accession: datasetName + "_biological_being_2_accession",
					},
					Attributes: &attributes,
				}, {
					ObjectType: metadata_models.ObjectType{
						Alias:     datasetName + "_specimen_3_alias",
						Accession: datasetName + "_specimen_3_accession",
					},
					ExtractedFromRef: metadata_models.Reference{
						Alias:     datasetName + "_biological_being_2_alias",
						Accession: datasetName + "_biological_being_2_accession",
					},
					Attributes: &attributes,
				},
			},
			Blocks: []metadata_models.Block{
				{
					ObjectType: metadata_models.ObjectType{
						Alias:     datasetName + "_block_1_alias",
						Accession: datasetName + "_block_1_accession",
					},
					SampledFromRef: []metadata_models.Reference{
						{
							Alias:     datasetName + "_specimen_1_alias",
							Accession: datasetName + "_specimen_1_accession",
						}, {
							Alias:     datasetName + "_specimen_2_alias",
							Accession: datasetName + "_specimen_2_accession",
						},
					},
					Attributes: &attributes,
				}, {
					ObjectType: metadata_models.ObjectType{
						Alias:     datasetName + "_block_2_alias",
						Accession: datasetName + "_block_2_accession",
					},
					SampledFromRef: []metadata_models.Reference{
						{
							Alias:     datasetName + "_specimen_3_alias",
							Accession: datasetName + "_specimen_3_accession",
						},
					},
					Attributes: &attributes,
				},
			},
			Slides: []metadata_models.Slide{
				{
					ObjectType: metadata_models.ObjectType{
						Alias:     datasetName + "_slide_1_alias",
						Accession: datasetName + "_slide_1_accession",
					},
					CreatedFromRef: metadata_models.Reference{
						Alias:     datasetName + "_block_1_alias",
						Accession: datasetName + "_block_1_accession",
					},
					StainingInformationRef: metadata_models.Reference{
						Alias:     datasetName + "_stain_1_alias",
						Accession: datasetName + "_stain_1_accession"},
					Attributes: &attributes,
				}, {
					ObjectType: metadata_models.ObjectType{
						Alias:     datasetName + "_slide_2_alias",
						Accession: datasetName + "_slide_2_accession",
					},
					CreatedFromRef: metadata_models.Reference{
						Alias:     datasetName + "_block_2_alias",
						Accession: datasetName + "_block_2_accession",
					},
					StainingInformationRef: metadata_models.Reference{
						Alias:     datasetName + "_stain_1_alias",
						Accession: datasetName + "_stain_1_accession"},
					Attributes: &attributes,
				},
			},
		},
		Staining: &metadata_models.StainingSet{
			Staining: []metadata_models.Staining{
				{
					ObjectType: metadata_models.ObjectType{
						Alias:     datasetName + "_stain_1_alias",
						Accession: datasetName + "_stain_1_accession",
					},
					ProcedureInformation: &metadata_models.Attributes{
						StringAttributes: []metadata_models.StringAttribute{
							{
								Tag: "attr_1",
								Value: &metadata_models.NullableString{
									Value: nil,
									Nil:   true,
								},
							},
						},
						NumericAttributes:     nil,
						MeasurementAttributes: nil,
						CodeAttributes:        nil,
						SetAttributes:         nil,
					},
					Stain:      nil,
					Attributes: &attributes,
				},
			},
		},
	}
}
