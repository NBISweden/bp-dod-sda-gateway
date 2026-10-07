package dataset_on_demand

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"connectrpc.com/authn"
	"connectrpc.com/connect"
	"github.com/NBISweden/bp-dod-sda-gateway/database"
	dodservice "github.com/NBISweden/bp-dod-sda-gateway/grpc_gen/go/public/v1"
	"github.com/NBISweden/bp-dod-sda-gateway/models"
	"github.com/NBISweden/bp-dod-sda-gateway/models/metadata_models"
	"github.com/NBISweden/bp-dod-sda-gateway/on_demand_dataset_metadata_file_handler"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockDodMetadataFileHandler struct {
	mock.Mock
}

func (m *mockDodMetadataFileHandler) RegisterOnDemandDataset(_ context.Context, dodDataset *models.OnDemandDataset) error {
	args := m.Called(dodDataset.Accession)

	return args.Error(0)
}

type mockDatabase struct {
	mock.Mock
}

func (m *mockDatabase) Commit() error {
	args := m.Called()

	return args.Error(0)
}

func (m *mockDatabase) Rollback() error {
	args := m.Called()

	return args.Error(0)
}

func (m *mockDatabase) BeginTransaction(_ context.Context) (database.Transaction, error) {
	args := m.Called()

	return args.Get(0).(*mockDatabase), args.Error(1)
}

func (m *mockDatabase) Close() error {
	args := m.Called()

	return args.Error(0)
}

func (m *mockDatabase) SchemaVersion() (uint, error) {
	args := m.Called()

	return args.Get(0).(uint), args.Error(1)
}

func (m *mockDatabase) Ping(ctx context.Context) error {
	args := m.Called()

	return args.Error(0)
}

func (m *mockDatabase) GetOriginDataset(_ context.Context, accession string) (*models.OriginDataset, error) {
	args := m.Called(accession)

	return args.Get(0).(*models.OriginDataset), args.Error(1)
}

func (m *mockDatabase) GetOriginDatasetAccessionFromImageAccession(_ context.Context, imageAccession string) (string, error) {
	args := m.Called(imageAccession)

	return args.String(0), args.Error(1)
}

func (m *mockDatabase) InsertOriginDataset(_ context.Context, originDataset *models.OriginDataset) error {
	args := m.Called(originDataset.Accession, originDataset.RemsWorkflowID, originDataset.RemsOrganisationID)

	return args.Error(0)
}

func (m *mockDatabase) InsertOnDemandDataset(_ context.Context, onDemandDataset *models.OnDemandDataset, imageAccessionHash string) error {
	args := m.Called(onDemandDataset.Accession, imageAccessionHash)

	return args.Error(0)
}

func (m *mockDatabase) InsertOnDemandDatasetImage(_ context.Context, onDemandDatasetAccession, imageAccession string) error {
	args := m.Called(onDemandDatasetAccession, imageAccession)

	return args.Error(0)
}

func (m *mockDatabase) InsertDatasetImage(_ context.Context, datasetAccession, imageAccession string) error {
	args := m.Called(datasetAccession, imageAccession)

	return args.Error(0)
}

func (m *mockDatabase) InsertImageFile(_ context.Context, datasetAccession, imageAccession, fileAccession, baseFileName string) error {
	args := m.Called(datasetAccession, imageAccession, fileAccession, baseFileName)

	return args.Error(0)
}

func (m *mockDatabase) ListUnreleasedOnDemandDatasetMetadataFiles(_ context.Context) (map[string]map[metadata_models.MetadataFileType]string, error) {
	args := m.Called()

	return args.Get(0).(map[string]map[metadata_models.MetadataFileType]string), args.Error(1)
}

func (m *mockDatabase) ListOnDemandDatasetImageFiles(_ context.Context, datasetAccession string) (map[string]map[string]string, error) {
	args := m.Called(datasetAccession)

	return args.Get(0).(map[string]map[string]string), args.Error(1)
}

func (m *mockDatabase) GetOnDemandDatasetRemsMetadata(_ context.Context, datasetAccession string) (*metadata_models.RemsSet, error) {
	args := m.Called(datasetAccession)

	return args.Get(0).(*metadata_models.RemsSet), args.Error(1)
}

func (m *mockDatabase) GetOnDemandDatasetAccessionFromImageAccessionsHash(_ context.Context, imageAccessionHash string) (string, error) {
	args := m.Called(imageAccessionHash)

	return args.String(0), args.Error(1)
}

func (m *mockDatabase) IsOnDemandDatasetReleased(_ context.Context, dodDatasetAccession string) (bool, error) {
	args := m.Called(dodDatasetAccession)

	return args.Bool(0), args.Error(1)
}

func (m *mockDatabase) SetOnDemandDatasetReleased(_ context.Context, datasetAccession string) error {
	args := m.Called(datasetAccession)

	return args.Error(0)
}

func TestRequestDatasetCreation(t *testing.T) {
	for _, tc := range []struct {
		name                             string
		user                             string
		imageAccessions                  []string
		mockDodMetadataFileHandlerOn     func() *mockDodMetadataFileHandler
		mockDodMetadataFileHandlerAssert func(*testing.T, *mockDodMetadataFileHandler)
		mockDatabaseOn                   func() *mockDatabase
		mockDatabaseAssert               func(*testing.T, *mockDatabase)
		expectedError                    error
		expectedNilResponse              bool
	}{
		{
			name:            "success",
			user:            "unit_test",
			imageAccessions: []string{"image_1", "image_2", "image_3"},
			mockDodMetadataFileHandlerOn: func() *mockDodMetadataFileHandler {
				mockDmfh := new(mockDodMetadataFileHandler)
				mockDmfh.On("RegisterOnDemandDataset", mock.Anything).Return(nil)

				return mockDmfh
			},
			mockDodMetadataFileHandlerAssert: func(t *testing.T, mockDmfh *mockDodMetadataFileHandler) {
				mockDmfh.AssertNumberOfCalls(t, "RegisterOnDemandDataset", 1)
			},
			mockDatabaseOn: func() *mockDatabase {
				mockDB := &mockDatabase{}
				mockDB.On("GetOnDemandDatasetAccessionFromImageAccessionsHash", mock.Anything).Return("", nil)
				mockDB.On("GetOriginDatasetAccessionFromImageAccession", "image_1").Return("dataset_1", nil)
				mockDB.On("GetOriginDatasetAccessionFromImageAccession", "image_2").Return("dataset_2", nil)
				mockDB.On("GetOriginDatasetAccessionFromImageAccession", "image_3").Return("dataset_3", nil)
				mockDB.On("GetOriginDataset", "dataset_1").Return(&models.OriginDataset{
					Accession:          "dataset_1",
					RemsWorkflowID:     2,
					RemsOrganisationID: "unit test",
					Dataset: &metadata_models.DatasetSet{
						Dataset: []metadata_models.Dataset{
							{
								ObjectType: metadata_models.ObjectType{
									Alias:     "dataset_1",
									Accession: "dataset_1",
								},
								ImageRef: []metadata_models.Reference{
									{
										Alias:     "image_1",
										Accession: "image_1",
									},
								},
							},
						},
					},
					Image: &metadata_models.ImageSet{
						Images: []metadata_models.Image{
							{
								ObjectType: metadata_models.ObjectType{
									Alias:     "image_1",
									Accession: "image_1",
								},
								ImageOf: metadata_models.Reference{
									Alias:     "slide_1",
									Accession: "slide_1",
								},
								Files: metadata_models.ImageFiles{
									Files: []metadata_models.ImageFile{
										{
											FileBaseType: metadata_models.FileBaseType{
												Filename:            "image_1_file_1",
												ChecksumMethod:      "unit_test",
												Checksum:            "123",
												UnencryptedChecksum: "",
											},
											FileType: "test",
										},
									},
								},
							},
						},
					},
					Annotation: nil,
					Observation: &metadata_models.ObservationSet{
						Observations: []metadata_models.Observation{},
					},
					Observer: nil,
					Policy: &metadata_models.PolicySet{
						Policies: []metadata_models.Policy{
							{
								Attributes: metadata_models.NullableAttributes{
									Value: &metadata_models.Attributes{
										StringAttributes: []metadata_models.StringAttribute{
											{
												Tag: "same_tag",
												Value: &metadata_models.NullableString{
													Value: new("same value"),
												},
											},
										},
									},
								},
							},
						},
					},
					Sample: &metadata_models.SampleSet{
						BiologicalBeings: []metadata_models.BiologicalBeing{
							{
								ObjectType: metadata_models.ObjectType{
									Alias:     "bio_being_1",
									Accession: "bio_being_1",
								},
							},
						},
						Cases: []metadata_models.Case{
							{
								ObjectType: metadata_models.ObjectType{
									Alias:     "case_1",
									Accession: "case_1",
								},
								BiologicalBeingRef: metadata_models.Reference{
									Alias:     "bio_being_1",
									Accession: "bio_being_1",
								},
							},
						},
						Specimens: []metadata_models.Specimen{
							{
								ObjectType: metadata_models.ObjectType{
									Alias:     "specimen_1",
									Accession: "specimen_1",
								},
								ExtractedFromRef: metadata_models.Reference{
									Alias:     "bio_being_1",
									Accession: "bio_being_1",
								},
							},
						},
						Blocks: []metadata_models.Block{
							{
								ObjectType: metadata_models.ObjectType{
									Alias:     "block_1",
									Accession: "block_1",
								},
								SampledFromRef: []metadata_models.Reference{
									{
										Alias:     "specimen_1",
										Accession: "specimen_1",
									},
								},
							},
						},
						Slides: []metadata_models.Slide{
							{
								ObjectType: metadata_models.ObjectType{
									Alias:     "slide_1",
									Accession: "slide_1",
								},
								CreatedFromRef: metadata_models.Reference{
									Alias:     "block_1",
									Accession: "block_1",
								},
								StainingInformationRef: metadata_models.Reference{
									Alias:     "stain_1",
									Accession: "stain_1",
								},
							},
						},
					},
					Staining: &metadata_models.StainingSet{
						Staining: []metadata_models.Staining{
							{
								ObjectType: metadata_models.ObjectType{
									Alias:     "stain_1",
									Accession: "stain_1",
								},
								ProcedureInformation: &metadata_models.Attributes{},
							},
						},
					},
				}, nil)
				mockDB.On("GetOriginDataset", "dataset_2").Return(&models.OriginDataset{
					Accession:          "dataset_2",
					RemsWorkflowID:     2,
					RemsOrganisationID: "unit test",
					Dataset: &metadata_models.DatasetSet{
						Dataset: []metadata_models.Dataset{
							{
								ObjectType: metadata_models.ObjectType{
									Alias:     "dataset_2",
									Accession: "dataset_2",
								},
								ImageRef: []metadata_models.Reference{
									{
										Alias:     "image_2",
										Accession: "image_2",
									},
								},
							},
						},
					},
					Image: &metadata_models.ImageSet{
						Images: []metadata_models.Image{
							{
								ObjectType: metadata_models.ObjectType{
									Alias:     "image_2",
									Accession: "image_2",
								},
								ImageOf: metadata_models.Reference{
									Alias:     "slide_2",
									Accession: "slide_2",
								},
								Files: metadata_models.ImageFiles{
									Files: []metadata_models.ImageFile{
										{
											FileBaseType: metadata_models.FileBaseType{
												Filename:            "image_2_file_1",
												ChecksumMethod:      "unit_test",
												Checksum:            "123",
												UnencryptedChecksum: "",
											},
											FileType: "test",
										},
									},
								},
							},
						},
					},
					Annotation: nil,
					Observation: &metadata_models.ObservationSet{
						Observations: []metadata_models.Observation{},
					},
					Observer: nil,
					Policy: &metadata_models.PolicySet{
						Policies: []metadata_models.Policy{
							{
								Attributes: metadata_models.NullableAttributes{
									Value: &metadata_models.Attributes{
										StringAttributes: []metadata_models.StringAttribute{
											{
												Tag: "same_tag",
												Value: &metadata_models.NullableString{
													Value: new("same value"),
												},
											},
										},
									},
								},
							},
						},
					},
					Sample: &metadata_models.SampleSet{
						BiologicalBeings: []metadata_models.BiologicalBeing{
							{
								ObjectType: metadata_models.ObjectType{
									Alias:     "bio_being_2",
									Accession: "bio_being_2",
								},
							},
						},
						Cases: []metadata_models.Case{
							{
								ObjectType: metadata_models.ObjectType{
									Alias:     "case_2",
									Accession: "case_2",
								},
								BiologicalBeingRef: metadata_models.Reference{
									Alias:     "bio_being_2",
									Accession: "bio_being_2",
								},
							},
						},
						Specimens: []metadata_models.Specimen{
							{
								ObjectType: metadata_models.ObjectType{
									Alias:     "specimen_2",
									Accession: "specimen_2",
								},
								ExtractedFromRef: metadata_models.Reference{
									Alias:     "bio_being_2",
									Accession: "bio_being_2",
								},
							},
						},
						Blocks: []metadata_models.Block{
							{
								ObjectType: metadata_models.ObjectType{
									Alias:     "block_2",
									Accession: "block_2",
								},
								SampledFromRef: []metadata_models.Reference{
									{
										Alias:     "specimen_2",
										Accession: "specimen_2",
									},
								},
							},
						},
						Slides: []metadata_models.Slide{
							{
								ObjectType: metadata_models.ObjectType{
									Alias:     "slide_2",
									Accession: "slide_2",
								},
								CreatedFromRef: metadata_models.Reference{
									Alias:     "block_2",
									Accession: "block_2",
								},
								StainingInformationRef: metadata_models.Reference{
									Alias:     "stain_2",
									Accession: "stain_2",
								},
							},
						},
					},
					Staining: &metadata_models.StainingSet{
						Staining: []metadata_models.Staining{
							{
								ObjectType: metadata_models.ObjectType{
									Alias:     "stain_2",
									Accession: "stain_2",
								},
								ProcedureInformation: &metadata_models.Attributes{},
							},
						},
					},
				}, nil)
				mockDB.On("GetOriginDataset", "dataset_3").Return(&models.OriginDataset{
					Accession:          "dataset_3",
					RemsWorkflowID:     2,
					RemsOrganisationID: "unit test",
					Dataset: &metadata_models.DatasetSet{
						Dataset: []metadata_models.Dataset{
							{
								ObjectType: metadata_models.ObjectType{
									Alias:     "dataset_3",
									Accession: "dataset_3",
								},
								ImageRef: []metadata_models.Reference{
									{
										Alias:     "image_3",
										Accession: "image_3",
									},
								},
							},
						},
					},
					Image: &metadata_models.ImageSet{
						Images: []metadata_models.Image{
							{
								ObjectType: metadata_models.ObjectType{
									Alias:     "image_3",
									Accession: "image_3",
								},
								ImageOf: metadata_models.Reference{
									Alias:     "slide_3",
									Accession: "slide_3",
								},
								Files: metadata_models.ImageFiles{
									Files: []metadata_models.ImageFile{
										{
											FileBaseType: metadata_models.FileBaseType{
												Filename:            "image_3_file_1",
												ChecksumMethod:      "unit_test",
												Checksum:            "123",
												UnencryptedChecksum: "",
											},
											FileType: "test",
										},
									},
								},
							},
						},
					},
					Annotation: nil,
					Observation: &metadata_models.ObservationSet{
						Observations: []metadata_models.Observation{},
					},
					Observer: nil,
					Policy: &metadata_models.PolicySet{
						Policies: []metadata_models.Policy{
							{
								Attributes: metadata_models.NullableAttributes{
									Value: &metadata_models.Attributes{
										StringAttributes: []metadata_models.StringAttribute{
											{
												Tag: "same_tag",
												Value: &metadata_models.NullableString{
													Value: new("same value"),
												},
											},
										},
									},
								},
							},
						},
					},
					Sample: &metadata_models.SampleSet{
						BiologicalBeings: []metadata_models.BiologicalBeing{
							{
								ObjectType: metadata_models.ObjectType{
									Alias:     "bio_being_3",
									Accession: "bio_being_3",
								},
							},
						},
						Cases: []metadata_models.Case{
							{
								ObjectType: metadata_models.ObjectType{
									Alias:     "case_3",
									Accession: "case_3",
								},
								BiologicalBeingRef: metadata_models.Reference{
									Alias:     "bio_being_3",
									Accession: "bio_being_3",
								},
							},
						},
						Specimens: []metadata_models.Specimen{
							{
								ObjectType: metadata_models.ObjectType{
									Alias:     "specimen_3",
									Accession: "specimen_3",
								},
								ExtractedFromRef: metadata_models.Reference{
									Alias:     "bio_being_3",
									Accession: "bio_being_3",
								},
							},
						},
						Blocks: []metadata_models.Block{
							{
								ObjectType: metadata_models.ObjectType{
									Alias:     "block_3",
									Accession: "block_3",
								},
								SampledFromRef: []metadata_models.Reference{
									{
										Alias:     "specimen_3",
										Accession: "specimen_3",
									},
								},
							},
						},
						Slides: []metadata_models.Slide{
							{
								ObjectType: metadata_models.ObjectType{
									Alias:     "slide_3",
									Accession: "slide_3",
								},
								CreatedFromRef: metadata_models.Reference{
									Alias:     "block_3",
									Accession: "block_3",
								},
								StainingInformationRef: metadata_models.Reference{
									Alias:     "stain_3",
									Accession: "stain_3",
								},
							},
						},
					},
					Staining: &metadata_models.StainingSet{
						Staining: []metadata_models.Staining{
							{
								ObjectType: metadata_models.ObjectType{
									Alias:     "stain_3",
									Accession: "stain_3",
								},
								ProcedureInformation: &metadata_models.Attributes{},
							},
						},
					},
				}, nil)

				mockDB.On("InsertOnDemandDataset", mock.Anything, mock.Anything).Return(nil)
				mockDB.On("InsertOnDemandDatasetImage", mock.Anything, mock.Anything).Return(nil)
				mockDB.On("BeginTransaction").Return(mockDB, nil)
				mockDB.On("Commit").Return(nil)
				mockDB.On("Rollback").Return(nil)

				return mockDB
			},
			mockDatabaseAssert: func(t *testing.T, mockDB *mockDatabase) {
				mockDB.AssertNumberOfCalls(t, "GetOnDemandDatasetAccessionFromImageAccessionsHash", 1)
				mockDB.AssertNumberOfCalls(t, "InsertOnDemandDataset", 1)
				mockDB.AssertNumberOfCalls(t, "InsertOnDemandDatasetImage", 3)

				mockDB.AssertNumberOfCalls(t, "Commit", 1)
				mockDB.AssertNumberOfCalls(t, "Rollback", 1)
			},
			expectedError:       nil,
			expectedNilResponse: false,
		}, {
			name:            "different_rems_workflow_id",
			user:            "unit_test",
			imageAccessions: []string{"image_1", "image_2", "image_3"},
			mockDodMetadataFileHandlerOn: func() *mockDodMetadataFileHandler {
				return new(mockDodMetadataFileHandler)
			},
			mockDodMetadataFileHandlerAssert: func(t *testing.T, mockDmfh *mockDodMetadataFileHandler) {
				mockDmfh.AssertNumberOfCalls(t, "RegisterOnDemandDataset", 0)
			},
			mockDatabaseOn: func() *mockDatabase {
				mockDB := &mockDatabase{}
				mockDB.On("GetOnDemandDatasetAccessionFromImageAccessionsHash", mock.Anything).Return("", nil)
				mockDB.On("GetOriginDatasetAccessionFromImageAccession", "image_1").Return("dataset_1", nil)
				mockDB.On("GetOriginDatasetAccessionFromImageAccession", "image_2").Return("dataset_2", nil)
				mockDB.On("GetOriginDatasetAccessionFromImageAccession", "image_3").Return("dataset_3", nil)
				mockDB.On("GetOriginDataset", "dataset_1").Return(&models.OriginDataset{
					Accession:          "dataset_1",
					RemsWorkflowID:     1,
					RemsOrganisationID: "unit test",
				}, nil)
				mockDB.On("GetOriginDataset", "dataset_2").Return(&models.OriginDataset{
					Accession:          "dataset_2",
					RemsWorkflowID:     2,
					RemsOrganisationID: "unit test",
				}, nil)
				mockDB.On("GetOriginDataset", "dataset_3").Return(&models.OriginDataset{
					Accession:          "dataset_3",
					RemsWorkflowID:     3,
					RemsOrganisationID: "unit test",
				}, nil)
				mockDB.On("BeginTransaction").Return(mockDB, nil)
				mockDB.On("Rollback").Return(nil)

				return mockDB
			},
			mockDatabaseAssert: func(t *testing.T, mockDB *mockDatabase) {
				mockDB.AssertNumberOfCalls(t, "GetOnDemandDatasetAccessionFromImageAccessionsHash", 1)
				mockDB.AssertNumberOfCalls(t, "InsertOnDemandDataset", 0)

				mockDB.AssertNumberOfCalls(t, "Commit", 0)
				mockDB.AssertNumberOfCalls(t, "Rollback", 1)
			},
			expectedError:       connect.NewError(connect.CodeInvalidArgument, errors.New("can not combine images from datasets originating from different DaCs")),
			expectedNilResponse: true,
		}, {
			name:            "different_policies_tags",
			user:            "unit_test",
			imageAccessions: []string{"image_1", "image_2", "image_3"},
			mockDodMetadataFileHandlerOn: func() *mockDodMetadataFileHandler {
				return new(mockDodMetadataFileHandler)
			},
			mockDodMetadataFileHandlerAssert: func(t *testing.T, mockDmfh *mockDodMetadataFileHandler) {
				mockDmfh.AssertNumberOfCalls(t, "RegisterOnDemandDataset", 0)
			},
			mockDatabaseOn: func() *mockDatabase {
				mockDB := &mockDatabase{}
				mockDB.On("GetOnDemandDatasetAccessionFromImageAccessionsHash", mock.Anything).Return("", nil)
				mockDB.On("GetOriginDatasetAccessionFromImageAccession", "image_1").Return("dataset_1", nil)
				mockDB.On("GetOriginDatasetAccessionFromImageAccession", "image_2").Return("dataset_2", nil)
				mockDB.On("GetOriginDatasetAccessionFromImageAccession", "image_3").Return("dataset_3", nil)
				mockDB.On("GetOriginDataset", "dataset_1").Return(&models.OriginDataset{
					Accession:          "dataset_1",
					RemsWorkflowID:     2,
					RemsOrganisationID: "unit test",
					Policy: &metadata_models.PolicySet{
						Policies: []metadata_models.Policy{
							{
								Attributes: metadata_models.NullableAttributes{
									Value: &metadata_models.Attributes{
										StringAttributes: []metadata_models.StringAttribute{
											{
												Tag: "diff_tag_1",
												Value: &metadata_models.NullableString{
													Value: new("unit_test"),
												},
											},
										},
									},
								},
							},
						},
					},
				}, nil)
				mockDB.On("GetOriginDataset", "dataset_2").Return(&models.OriginDataset{
					Accession:          "dataset_2",
					RemsWorkflowID:     2,
					RemsOrganisationID: "unit test",
					Policy: &metadata_models.PolicySet{
						Policies: []metadata_models.Policy{
							{
								Attributes: metadata_models.NullableAttributes{
									Value: &metadata_models.Attributes{
										StringAttributes: []metadata_models.StringAttribute{
											{
												Tag: "diff_tag_2",
												Value: &metadata_models.NullableString{
													Value: new("unit_test"),
												},
											},
										},
									},
								},
							},
						},
					},
				}, nil)
				mockDB.On("GetOriginDataset", "dataset_3").Return(&models.OriginDataset{
					Accession:          "dataset_3",
					RemsWorkflowID:     2,
					RemsOrganisationID: "unit test",
					Policy: &metadata_models.PolicySet{
						Policies: []metadata_models.Policy{
							{
								Attributes: metadata_models.NullableAttributes{
									Value: &metadata_models.Attributes{
										StringAttributes: []metadata_models.StringAttribute{
											{
												Tag: "diff_tag_3",
												Value: &metadata_models.NullableString{
													Value: new("unit_test"),
												},
											},
										},
									},
								},
							},
						},
					},
				}, nil)
				mockDB.On("BeginTransaction").Return(mockDB, nil)
				mockDB.On("Rollback").Return(nil)

				return mockDB
			},
			mockDatabaseAssert: func(t *testing.T, mockDB *mockDatabase) {
				mockDB.AssertNumberOfCalls(t, "GetOnDemandDatasetAccessionFromImageAccessionsHash", 1)
				mockDB.AssertNumberOfCalls(t, "InsertOnDemandDataset", 0)

				mockDB.AssertNumberOfCalls(t, "Commit", 0)
				mockDB.AssertNumberOfCalls(t, "Rollback", 1)
			},
			expectedError:       connect.NewError(connect.CodeInvalidArgument, errors.New("can not combine images from datasets with different terms of use")),
			expectedNilResponse: true,
		}, {
			name:            "different_policies_values",
			user:            "unit_test",
			imageAccessions: []string{"image_1", "image_2", "image_3"},
			mockDodMetadataFileHandlerOn: func() *mockDodMetadataFileHandler {
				return new(mockDodMetadataFileHandler)
			},
			mockDodMetadataFileHandlerAssert: func(t *testing.T, mockDmfh *mockDodMetadataFileHandler) {
				mockDmfh.AssertNumberOfCalls(t, "RegisterOnDemandDataset", 0)
			},
			mockDatabaseOn: func() *mockDatabase {
				mockDB := &mockDatabase{}
				mockDB.On("GetOnDemandDatasetAccessionFromImageAccessionsHash", mock.Anything).Return("", nil)
				mockDB.On("GetOriginDatasetAccessionFromImageAccession", "image_1").Return("dataset_1", nil)
				mockDB.On("GetOriginDatasetAccessionFromImageAccession", "image_2").Return("dataset_2", nil)
				mockDB.On("GetOriginDatasetAccessionFromImageAccession", "image_3").Return("dataset_3", nil)
				mockDB.On("GetOriginDataset", "dataset_1").Return(&models.OriginDataset{
					Accession:          "dataset_1",
					RemsWorkflowID:     2,
					RemsOrganisationID: "unit test",
					Policy: &metadata_models.PolicySet{
						Policies: []metadata_models.Policy{
							{
								Attributes: metadata_models.NullableAttributes{
									Value: &metadata_models.Attributes{
										StringAttributes: []metadata_models.StringAttribute{
											{
												Tag: "same_tag",
												Value: &metadata_models.NullableString{
													Value: new("diff value 1"),
												},
											},
										},
									},
								},
							},
						},
					},
				}, nil)
				mockDB.On("GetOriginDataset", "dataset_2").Return(&models.OriginDataset{
					Accession:          "dataset_2",
					RemsWorkflowID:     2,
					RemsOrganisationID: "unit test",
					Policy: &metadata_models.PolicySet{
						Policies: []metadata_models.Policy{
							{
								Attributes: metadata_models.NullableAttributes{
									Value: &metadata_models.Attributes{
										StringAttributes: []metadata_models.StringAttribute{
											{
												Tag: "same_tag",
												Value: &metadata_models.NullableString{
													Value: new("diff value 2"),
												},
											},
										},
									},
								},
							},
						},
					},
				}, nil)
				mockDB.On("GetOriginDataset", "dataset_3").Return(&models.OriginDataset{
					Accession:          "dataset_3",
					RemsWorkflowID:     2,
					RemsOrganisationID: "unit test",
					Policy: &metadata_models.PolicySet{
						Policies: []metadata_models.Policy{
							{
								Attributes: metadata_models.NullableAttributes{
									Value: &metadata_models.Attributes{
										StringAttributes: []metadata_models.StringAttribute{
											{
												Tag: "same_tag",
												Value: &metadata_models.NullableString{
													Value: new("diff value 3"),
												},
											},
										},
									},
								},
							},
						},
					},
				}, nil)
				mockDB.On("BeginTransaction").Return(mockDB, nil)
				mockDB.On("Rollback").Return(nil)

				return mockDB
			},
			mockDatabaseAssert: func(t *testing.T, mockDB *mockDatabase) {
				mockDB.AssertNumberOfCalls(t, "GetOnDemandDatasetAccessionFromImageAccessionsHash", 1)
				mockDB.AssertNumberOfCalls(t, "InsertOnDemandDataset", 0)

				mockDB.AssertNumberOfCalls(t, "Commit", 0)
				mockDB.AssertNumberOfCalls(t, "Rollback", 1)
			},
			expectedError:       connect.NewError(connect.CodeInvalidArgument, errors.New("can not combine images from datasets with different terms of use")),
			expectedNilResponse: true,
		}, {
			name:            "already_exists",
			user:            "unit_test",
			imageAccessions: []string{"image_1", "image_2", "image_3"},
			mockDodMetadataFileHandlerOn: func() *mockDodMetadataFileHandler {
				return new(mockDodMetadataFileHandler)
			},
			mockDodMetadataFileHandlerAssert: func(t *testing.T, mockDmfh *mockDodMetadataFileHandler) {
				mockDmfh.AssertNumberOfCalls(t, "RegisterOnDemandDataset", 0)
			},
			mockDatabaseOn: func() *mockDatabase {
				mockDB := &mockDatabase{}
				expectedImageHash := hashImageAccessions([]string{"image_1", "image_2", "image_3"})
				mockDB.On("GetOnDemandDatasetAccessionFromImageAccessionsHash", expectedImageHash).Return("existing_id", nil)

				return mockDB
			},
			mockDatabaseAssert: func(t *testing.T, mockDB *mockDatabase) {
				mockDB.AssertNumberOfCalls(t, "GetOnDemandDatasetAccessionFromImageAccessionsHash", 1)
			},
			expectedError:       nil,
			expectedNilResponse: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mockDB := tc.mockDatabaseOn()
			mockDmfh := tc.mockDodMetadataFileHandlerOn()

			on_demand_dataset_metadata_file_handler.RegisterDodMetadataFileHandler(mockDmfh)
			database.RegisterDatabase(mockDB)
			testDodServiceImpl := &serviceImpl{}
			testToken := jwt.New()
			_ = testToken.Set("sub", "unit_test_user")
			ctx := authn.SetInfo(context.Background(), testToken)
			rsp, err := testDodServiceImpl.RequestDatasetCreation(ctx, connect.NewRequest(&dodservice.RequestDatasetCreationRequest{ImageAccessions: tc.imageAccessions}))
			assert.Equal(t, tc.expectedError, err)
			assert.Equal(t, tc.expectedNilResponse, rsp == nil)

			tc.mockDatabaseAssert(t, mockDB)
			tc.mockDodMetadataFileHandlerAssert(t, mockDmfh)
		})
	}
}

func TestGetOnDemandDatasetStatus(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		onDemandDatasetAccession string
		mockDatabaseOn           func() *mockDatabase
		mockDatabaseAssert       func(*testing.T, *mockDatabase)
		expectedError            error
		expectedResponse         *connect.Response[dodservice.GetOnDemandDatasetStatusResponse]
	}{
		{
			name:                     "not_found",
			onDemandDatasetAccession: "not_found",
			mockDatabaseOn: func() *mockDatabase {
				mockDB := new(mockDatabase)
				mockDB.On("IsOnDemandDatasetReleased", "not_found").Return(false, sql.ErrNoRows)

				return mockDB
			},
			mockDatabaseAssert: func(t *testing.T, mockDB *mockDatabase) {
				mockDB.AssertNumberOfCalls(t, "IsOnDemandDatasetReleased", 1)
			},
			expectedError:    connect.NewError(connect.CodeNotFound, nil),
			expectedResponse: nil,
		}, {
			name:                     "db_error",
			onDemandDatasetAccession: "db_error",
			mockDatabaseOn: func() *mockDatabase {
				mockDB := new(mockDatabase)
				mockDB.On("IsOnDemandDatasetReleased", "db_error").Return(false, errors.New("db error"))

				return mockDB
			},
			mockDatabaseAssert: func(t *testing.T, mockDB *mockDatabase) {
				mockDB.AssertNumberOfCalls(t, "IsOnDemandDatasetReleased", 1)
			},
			expectedError:    connect.NewError(connect.CodeInternal, nil),
			expectedResponse: nil,
		}, {
			name:                     "found_and_released",
			onDemandDatasetAccession: "found_and_released",
			mockDatabaseOn: func() *mockDatabase {
				mockDB := new(mockDatabase)
				mockDB.On("IsOnDemandDatasetReleased", "found_and_released").Return(true, nil)

				return mockDB
			},
			mockDatabaseAssert: func(t *testing.T, mockDB *mockDatabase) {
				mockDB.AssertNumberOfCalls(t, "IsOnDemandDatasetReleased", 1)
			},
			expectedError: nil,
			expectedResponse: connect.NewResponse(&dodservice.GetOnDemandDatasetStatusResponse{
				Status: dodservice.GetOnDemandDatasetStatusResponse_STATUS_RELEASED,
			}),
		}, {
			name:                     "found_and_pending",
			onDemandDatasetAccession: "found_and_pending",
			mockDatabaseOn: func() *mockDatabase {
				mockDB := new(mockDatabase)
				mockDB.On("IsOnDemandDatasetReleased", "found_and_pending").Return(false, nil)

				return mockDB
			},
			mockDatabaseAssert: func(t *testing.T, mockDB *mockDatabase) {
				mockDB.AssertNumberOfCalls(t, "IsOnDemandDatasetReleased", 1)
			},
			expectedError: nil,
			expectedResponse: connect.NewResponse(&dodservice.GetOnDemandDatasetStatusResponse{
				Status: dodservice.GetOnDemandDatasetStatusResponse_STATUS_CREATING,
			}),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mockDB := tc.mockDatabaseOn()

			database.RegisterDatabase(mockDB)
			testDodServiceImpl := &serviceImpl{}
			rsp, err := testDodServiceImpl.GetOnDemandDatasetStatus(context.Background(), connect.NewRequest(&dodservice.GetOnDemandDatasetStatusRequest{OnDemandDatasetAccession: tc.onDemandDatasetAccession}))
			assert.Equal(t, tc.expectedError, err)
			assert.Equal(t, tc.expectedResponse, rsp)

			tc.mockDatabaseAssert(t, mockDB)
		})
	}
}

func TestHashImageAccessions(t *testing.T) {
	for _, tc := range []struct {
		name           string
		inputA, inputB []string
		expectedEqual  bool
	}{
		{
			name:          "same_input",
			inputA:        []string{"test", "test123", "test321"},
			inputB:        []string{"test", "test123", "test321"},
			expectedEqual: true,
		},
		{
			name:          "same_input_differnt_order",
			inputA:        []string{"test321", "test123", "test"},
			inputB:        []string{"test", "test123", "test321"},
			expectedEqual: true,
		},
		{
			name:          "same_input_with_duplicates",
			inputA:        []string{"test", "test123", "test321", "test", "test123", "test321"},
			inputB:        []string{"test", "test123", "test321"},
			expectedEqual: true,
		},
		{
			name:          "different_input",
			inputA:        []string{"test", "test123", "test321"},
			inputB:        []string{"test123", "test321"},
			expectedEqual: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedEqual, hashImageAccessions(tc.inputA) == hashImageAccessions(tc.inputB))
		})
	}
}
