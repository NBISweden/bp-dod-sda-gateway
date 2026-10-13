package dataset_on_demand_admin

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/NBISweden/bp-dod-sda-gateway/database"
	dodservice "github.com/NBISweden/bp-dod-sda-gateway/grpc_gen/go/private/v1"
	"github.com/NBISweden/bp-dod-sda-gateway/models"
	"github.com/NBISweden/bp-dod-sda-gateway/models/metadata_models"
	"github.com/NBISweden/bp-dod-sda-gateway/origin_dataset_file_loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockOriginDatasetLoader struct {
	mock.Mock
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

func (m *mockOriginDatasetLoader) UnmarshalFileToXml(_ context.Context, file *origin_dataset_file_loader.FileInfo, dst any) error {
	args := m.Called(file.DatasetAccession, file.MetadataFileType.String())

	switch file.MetadataFileType {
	case metadata_models.MetadataFileTypeDataset:
		*(dst).(*metadata_models.DatasetSet) = *(args.Get(0).(*metadata_models.DatasetSet))
	case metadata_models.MetadataFileTypeImage:
		*(dst).(*metadata_models.ImageSet) = *(args.Get(0).(*metadata_models.ImageSet))
	case metadata_models.MetadataFileTypeAnnotation:
		*(dst).(*metadata_models.AnnotationSet) = *(args.Get(0).(*metadata_models.AnnotationSet))
	case metadata_models.MetadataFileTypeObservation:
		*(dst).(*metadata_models.ObservationSet) = *(args.Get(0).(*metadata_models.ObservationSet))
	case metadata_models.MetadataFileTypeObserver:
		*(dst).(*metadata_models.ObserverSet) = *(args.Get(0).(*metadata_models.ObserverSet))
	case metadata_models.MetadataFileTypePolicy:
		*(dst).(*metadata_models.PolicySet) = *(args.Get(0).(*metadata_models.PolicySet))
	case metadata_models.MetadataFileTypeRems:
		*(dst).(*metadata_models.RemsSet) = *(args.Get(0).(*metadata_models.RemsSet))
	case metadata_models.MetadataFileTypeSample:
		*(dst).(*metadata_models.SampleSet) = *(args.Get(0).(*metadata_models.SampleSet))
	case metadata_models.MetadataFileTypeStaining:
		*(dst).(*metadata_models.StainingSet) = *(args.Get(0).(*metadata_models.StainingSet))
	default:
		return errors.New("unknown metadata file type")
	}

	return args.Error(1)
}

func (m *mockOriginDatasetLoader) ListDatasetFiles(_ context.Context, datasetAccession string) ([]*origin_dataset_file_loader.FileInfo, error) {
	args := m.Called(datasetAccession)

	return args.Get(0).([]*origin_dataset_file_loader.FileInfo), args.Error(1)
}

func (m *mockOriginDatasetLoader) GetRemsWorkFlowIDAndOrganisationID(_ context.Context, datasetAccession string) (int, string, error) {
	args := m.Called(datasetAccession)

	return args.Int(0), args.String(1), args.Error(2)
}

func (m *mockOriginDatasetLoader) Ping(_ context.Context) error {
	args := m.Called()

	return args.Error(0)
}

func (m *mockOriginDatasetLoader) Close() error {
	args := m.Called()

	return args.Error(0)
}

func TestNewOriginDataset(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		originDatasetAccession string
		mockODLOn              func() *mockOriginDatasetLoader
		mockODLAssert          func(*testing.T, *mockOriginDatasetLoader)
		mockDatabaseOn         func() *mockDatabase
		mockDatabaseAssert     func(*testing.T, *mockDatabase)
		expectedError          error
		expectedNilResponse    bool
	}{
		{
			name:                   "not_found",
			originDatasetAccession: "not_found",
			mockODLOn: func() *mockOriginDatasetLoader {
				mockODL := new(mockOriginDatasetLoader)

				mockODL.On("ListDatasetFiles", "not_found").Return([]*origin_dataset_file_loader.FileInfo{}, nil)

				return mockODL
			},
			mockODLAssert: func(t *testing.T, mockODL *mockOriginDatasetLoader) {
				mockODL.AssertNumberOfCalls(t, "ListDatasetFiles", 1)
			},
			mockDatabaseOn: func() *mockDatabase {
				return new(mockDatabase)
			},
			mockDatabaseAssert: func(t *testing.T, mockDB *mockDatabase) {
				mockDB.AssertNumberOfCalls(t, "BeginTransaction", 0)
			},
			expectedError:       connect.NewError(connect.CodeNotFound, nil),
			expectedNilResponse: true,
		}, {
			name:                   "missing_rems_metadata",
			originDatasetAccession: "missing_rems_metadata",
			mockODLOn: func() *mockOriginDatasetLoader {
				mockODL := new(mockOriginDatasetLoader)

				mockODL.On("ListDatasetFiles", "missing_rems_metadata").Return([]*origin_dataset_file_loader.FileInfo{
					{
						Accession:        "dataset",
						Path:             "dataset",
						DatasetAccession: "missing_rems_metadata",
						MetadataFileType: metadata_models.MetadataFileTypeDataset,
					},
				}, nil)

				mockODL.On("UnmarshalFileToXml", "missing_rems_metadata", "dataset").Return(&metadata_models.DatasetSet{}, nil)
				mockODL.On("GetRemsWorkFlowIDAndOrganisationID", "missing_rems_metadata").Return(-1, "", nil)

				return mockODL
			},
			mockODLAssert: func(t *testing.T, mockODL *mockOriginDatasetLoader) {
				mockODL.AssertNumberOfCalls(t, "ListDatasetFiles", 1)
				mockODL.AssertNumberOfCalls(t, "UnmarshalFileToXml", 1)
				mockODL.AssertNumberOfCalls(t, "GetRemsWorkFlowIDAndOrganisationID", 1)
			},
			mockDatabaseOn: func() *mockDatabase {
				return new(mockDatabase)
			},
			mockDatabaseAssert: func(t *testing.T, mockDB *mockDatabase) {
				mockDB.AssertNumberOfCalls(t, "BeginTransaction", 0)
			},
			expectedError:       connect.NewError(connect.CodeFailedPrecondition, errors.New("rems.xml not found")),
			expectedNilResponse: true,
		}, {
			name:                   "missing_metadata_file",
			originDatasetAccession: "missing_metadata_file",
			mockODLOn: func() *mockOriginDatasetLoader {
				mockODL := new(mockOriginDatasetLoader)

				mockODL.On("ListDatasetFiles", "missing_metadata_file").Return([]*origin_dataset_file_loader.FileInfo{
					{
						Accession:        "dataset",
						Path:             "dataset",
						DatasetAccession: "missing_metadata_file",
						MetadataFileType: metadata_models.MetadataFileTypeDataset,
					},
				}, nil)

				mockODL.On("UnmarshalFileToXml", "missing_metadata_file", "dataset").Return(&metadata_models.DatasetSet{}, nil)
				mockODL.On("GetRemsWorkFlowIDAndOrganisationID", "missing_metadata_file").Return(1, "test-org", nil)

				return mockODL
			},
			mockODLAssert: func(t *testing.T, mockODL *mockOriginDatasetLoader) {
				mockODL.AssertNumberOfCalls(t, "ListDatasetFiles", 1)
				mockODL.AssertNumberOfCalls(t, "UnmarshalFileToXml", 1)
				mockODL.AssertNumberOfCalls(t, "GetRemsWorkFlowIDAndOrganisationID", 1)
			},
			mockDatabaseOn: func() *mockDatabase {
				return new(mockDatabase)
			},
			mockDatabaseAssert: func(t *testing.T, mockDB *mockDatabase) {
				mockDB.AssertNumberOfCalls(t, "BeginTransaction", 0)
			},
			expectedError:       connect.NewError(connect.CodeFailedPrecondition, errors.New("image.xml not found")),
			expectedNilResponse: true,
		}, {
			name:                   "success",
			originDatasetAccession: "success",
			mockODLOn: func() *mockOriginDatasetLoader {
				mockODL := new(mockOriginDatasetLoader)

				mockODL.On("ListDatasetFiles", "success").Return([]*origin_dataset_file_loader.FileInfo{
					{
						Accession:        "dataset",
						Path:             "dataset",
						DatasetAccession: "success",
						MetadataFileType: metadata_models.MetadataFileTypeDataset,
					}, {
						Accession:        "image",
						Path:             "image",
						DatasetAccession: "success",
						MetadataFileType: metadata_models.MetadataFileTypeImage,
					}, {
						Accession:        "observation",
						Path:             "observation",
						DatasetAccession: "success",
						MetadataFileType: metadata_models.MetadataFileTypeObservation,
					}, {
						Accession:        "policy",
						Path:             "policy",
						DatasetAccession: "success",
						MetadataFileType: metadata_models.MetadataFileTypePolicy,
					}, {
						Accession:        "sample",
						Path:             "sample",
						DatasetAccession: "success",
						MetadataFileType: metadata_models.MetadataFileTypeSample,
					}, {
						Accession:        "staining",
						Path:             "staining",
						DatasetAccession: "success",
						MetadataFileType: metadata_models.MetadataFileTypeStaining,
					}, {
						Accession:        "image_1/file1",
						Path:             "image_1/file1",
						DatasetAccession: "success",
						MetadataFileType: metadata_models.MetadataFileTypeInvalid,
					}, {
						Accession:        "image_1/file2",
						Path:             "image_1/file2",
						DatasetAccession: "success",
						MetadataFileType: metadata_models.MetadataFileTypeInvalid,
					}, {
						Accession:        "image_2/file1",
						Path:             "image_2/file1",
						DatasetAccession: "success",
						MetadataFileType: metadata_models.MetadataFileTypeInvalid,
					}, {
						Accession:        "image_2/file2",
						Path:             "image_2/file2",
						DatasetAccession: "success",
						MetadataFileType: metadata_models.MetadataFileTypeInvalid,
					}, {
						Accession:        "image_3/file1",
						Path:             "image_3/file1",
						DatasetAccession: "success",
						MetadataFileType: metadata_models.MetadataFileTypeInvalid,
					}, {
						Accession:        "image_3/file2",
						Path:             "image_3/file2",
						DatasetAccession: "success",
						MetadataFileType: metadata_models.MetadataFileTypeInvalid,
					}, {
						Accession:        "image_4/file1",
						Path:             "image_4/file1",
						DatasetAccession: "success",
						MetadataFileType: metadata_models.MetadataFileTypeInvalid,
					}, {
						Accession:        "image_4/file2",
						Path:             "image_4/file2",
						DatasetAccession: "success",
						MetadataFileType: metadata_models.MetadataFileTypeInvalid,
					},
				}, nil)

				mockODL.On("UnmarshalFileToXml", "success", "dataset").Return(&metadata_models.DatasetSet{
					Dataset: []metadata_models.Dataset{
						{
							ObjectType: metadata_models.ObjectType{
								Alias:     "dataset",
								Accession: "dataset",
							},
						},
					},
				}, nil)
				mockODL.On("UnmarshalFileToXml", "success", "image").Return(&metadata_models.ImageSet{
					Images: []metadata_models.Image{
						{
							ObjectType: metadata_models.ObjectType{
								Alias:     "image_1",
								Accession: "image_1",
							},
							Files: metadata_models.ImageFiles{
								Files: []metadata_models.ImageFile{
									{
										FileBaseType: metadata_models.FileBaseType{
											Filename: "image_1/file1",
										},
									}, {
										FileBaseType: metadata_models.FileBaseType{
											Filename: "image_1/file2",
										},
									},
								},
							},
						},
						{
							ObjectType: metadata_models.ObjectType{
								Alias:     "image_2",
								Accession: "image_2",
							}, Files: metadata_models.ImageFiles{
								Files: []metadata_models.ImageFile{
									{
										FileBaseType: metadata_models.FileBaseType{
											Filename: "image_2/file1",
										},
									}, {
										FileBaseType: metadata_models.FileBaseType{
											Filename: "image_2/file2",
										},
									},
								},
							},
						},
						{
							ObjectType: metadata_models.ObjectType{
								Alias:     "image_3",
								Accession: "image_3",
							},
							Files: metadata_models.ImageFiles{
								Files: []metadata_models.ImageFile{
									{
										FileBaseType: metadata_models.FileBaseType{
											Filename: "image_3/file1",
										},
									}, {
										FileBaseType: metadata_models.FileBaseType{
											Filename: "image_3/file2",
										},
									},
								},
							},
						},
						{
							ObjectType: metadata_models.ObjectType{
								Alias:     "image_4",
								Accession: "image_4",
							},
							Files: metadata_models.ImageFiles{
								Files: []metadata_models.ImageFile{
									{
										FileBaseType: metadata_models.FileBaseType{
											Filename: "image_4/file1",
										},
									}, {
										FileBaseType: metadata_models.FileBaseType{
											Filename: "image_4/file2",
										},
									},
								},
							},
						},
					},
				}, nil)
				mockODL.On("UnmarshalFileToXml", "success", "observation").Return(&metadata_models.ObservationSet{}, nil)
				mockODL.On("UnmarshalFileToXml", "success", "policy").Return(&metadata_models.PolicySet{}, nil)
				mockODL.On("UnmarshalFileToXml", "success", "sample").Return(&metadata_models.SampleSet{}, nil)
				mockODL.On("UnmarshalFileToXml", "success", "staining").Return(&metadata_models.StainingSet{}, nil)
				mockODL.On("GetRemsWorkFlowIDAndOrganisationID", "success").Return(1, "test-org", nil)

				return mockODL
			},
			mockODLAssert: func(t *testing.T, mockODL *mockOriginDatasetLoader) {
				mockODL.AssertNumberOfCalls(t, "ListDatasetFiles", 1)
				mockODL.AssertNumberOfCalls(t, "UnmarshalFileToXml", 6)
				mockODL.AssertNumberOfCalls(t, "GetRemsWorkFlowIDAndOrganisationID", 1)
			},
			mockDatabaseOn: func() *mockDatabase {
				mockDB := &mockDatabase{}
				mockDB.On("BeginTransaction").Return(mockDB, nil)
				mockDB.On("InsertOriginDataset", "success", 1, "test-org").Return(nil)
				mockDB.On("InsertDatasetImage", mock.Anything, mock.Anything).Return(nil)
				mockDB.On("InsertImageFile", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockDB.On("Commit").Return(nil)
				mockDB.On("Rollback").Return(nil)

				return mockDB
			},
			mockDatabaseAssert: func(t *testing.T, mockDB *mockDatabase) {
				mockDB.AssertNumberOfCalls(t, "BeginTransaction", 1)
				mockDB.AssertNumberOfCalls(t, "InsertOriginDataset", 1)
				mockDB.AssertNumberOfCalls(t, "InsertDatasetImage", 4)
				mockDB.AssertNumberOfCalls(t, "InsertImageFile", 8)
				mockDB.AssertNumberOfCalls(t, "Commit", 1)
				mockDB.AssertNumberOfCalls(t, "Rollback", 1)
			},
			expectedError:       nil,
			expectedNilResponse: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mockODL := tc.mockODLOn()
			mockDB := tc.mockDatabaseOn()

			database.RegisterDatabase(mockDB)
			testDodServiceImpl := serviceImpl{
				originDatasetFileLoader: mockODL,
			}
			rsp, err := testDodServiceImpl.NewOriginDataset(context.Background(), connect.NewRequest(&dodservice.NewOriginDatasetRequest{DatasetAccession: tc.originDatasetAccession}))
			assert.Equal(t, tc.expectedError, err)
			assert.Equal(t, tc.expectedNilResponse, rsp == nil)

			tc.mockODLAssert(t, mockODL)
			tc.mockDatabaseAssert(t, mockDB)
		})
	}
}
