package origin_dataset_sync

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	dodservice "github.com/NBISweden/bp-dod-sda-gateway/grpc_gen/go/private/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockSyncer struct {
	mock.Mock
}

func (m *mockSyncer) triggerSync(ctx context.Context) {
	_ = m.Called()
}

func (m *mockSyncer) syncOriginDataset(_ context.Context, datasetAccession string) error {
	args := m.Called(datasetAccession)

	return args.Error(0)
}

func TestSyncOriginDataset(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		originDatasetAccession string
		mockSyncer             func() *mockSyncer
		expectedError          error
		expectedNilResponse    bool
	}{
		{
			name:                   "error",
			originDatasetAccession: "error",
			mockSyncer: func() *mockSyncer {
				mockODL := new(mockSyncer)

				mockODL.On("syncOriginDataset", "error").Return(errors.New("not found")).Once()

				return mockODL
			},
			expectedError:       connect.NewError(connect.CodeInternal, errors.New("not found")),
			expectedNilResponse: true,
		}, {
			name:                   "success",
			originDatasetAccession: "success",
			mockSyncer: func() *mockSyncer {
				mockODL := new(mockSyncer)

				mockODL.On("syncOriginDataset", "success").Return(nil).Once()

				return mockODL
			},
			expectedError:       nil,
			expectedNilResponse: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ms := tc.mockSyncer()

			testDodServiceImpl := serviceImpl{
				syncer: ms,
			}
			rsp, err := testDodServiceImpl.SyncOriginDataset(context.Background(), connect.NewRequest(&dodservice.SyncOriginDatasetRequest{DatasetAccession: tc.originDatasetAccession}))
			assert.Equal(t, tc.expectedError, err)
			assert.Equal(t, tc.expectedNilResponse, rsp == nil)

			ms.AssertExpectations(t)
		})
	}
}

func TestTriggerOriginDatasetsSync(t *testing.T) {
	for _, tc := range []struct {
		name       string
		mockSyncer func() *mockSyncer
	}{
		{
			name: "success",
			mockSyncer: func() *mockSyncer {
				mockODL := new(mockSyncer)

				mockODL.On("triggerSync").Once()

				return mockODL
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ms := tc.mockSyncer()

			testDodServiceImpl := serviceImpl{
				syncer: ms,
			}
			rsp, err := testDodServiceImpl.TriggerOriginDatasetsSync(context.Background(), connect.NewRequest(&dodservice.TriggerOriginDatasetsSyncRequest{}))
			assert.Equal(t, nil, err)
			assert.NotNil(t, rsp)

			ms.AssertExpectations(t)
		})
	}
}
