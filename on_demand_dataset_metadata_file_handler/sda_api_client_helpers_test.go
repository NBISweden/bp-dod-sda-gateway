package on_demand_dataset_metadata_file_handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/NBISweden/bp-dod-sda-gateway/models/metadata_models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockRoundTripper struct {
	requestsSeen []datasetCreateReq

	mock.Mock
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	var parsedReq datasetCreateReq
	if err := json.NewDecoder(req.Body).Decode(&parsedReq); err != nil {
		return nil, err
	}
	m.requestsSeen = append(m.requestsSeen, parsedReq)

	args := m.Called(req.Method, req.URL.Path)

	return args.Get(0).(*http.Response), args.Error(1)
}

func TestTriggerDatasetCreation(t *testing.T) {
	type testCase struct {
		name                        string
		datasetAccessionId          string
		imageAccessionFileNames     map[string]map[string]string
		metadataFiles               map[metadata_models.MetadataFileType]string
		newMockRoundTripper         func() *mockRoundTripper
		mockRoundTripperAssert      func(*testing.T, *mockRoundTripper)
		datasetCreateFilesBatchSize int
		expectedError               error
	}

	for _, tc := range []testCase{
		{
			name:                    "only_metadata_files_one_call",
			datasetAccessionId:      "123",
			imageAccessionFileNames: nil,
			metadataFiles: map[metadata_models.MetadataFileType]string{
				metadata_models.MetadataFileTypeDataset:     "1",
				metadata_models.MetadataFileTypeImage:       "2",
				metadata_models.MetadataFileTypeObservation: "3",
			},
			newMockRoundTripper: func() *mockRoundTripper {
				mrt := &mockRoundTripper{}

				mrt.On("RoundTrip", "POST", "dataset/create").Return(&http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     make(http.Header),
				}, nil).Once()

				return mrt
			},
			mockRoundTripperAssert: func(t *testing.T, mrt *mockRoundTripper) {
				mrt.AssertExpectations(t)
				assert.Len(t, mrt.requestsSeen, 1)
				assert.ElementsMatch(t, mrt.requestsSeen[0].FileAccessionIDs, []string{"1", "2", "3"})
				assert.Equal(t, mrt.requestsSeen[0].FileDownloadPaths, map[string]string{
					"1": "METADATA/dataset.xml.c4gh",
					"2": "METADATA/image.xml.c4gh",
					"3": "METADATA/observation.xml.c4gh",
				})
			},
			datasetCreateFilesBatchSize: 3,
			expectedError:               nil,
		}, {
			name:               "no_metadata_files_multiple_calls",
			datasetAccessionId: "123",
			imageAccessionFileNames: map[string]map[string]string{
				"image_1": {
					"image_1_file_1": "path_to_image_1_file_1",
					"image_1_file_2": "path_to_image_1_file_2",
					"image_1_file_3": "path_to_image_1_file_3",
				},
				"image_2": {
					"image_2_file_1": "path_to_image_2_file_1",
					"image_2_file_2": "path_to_image_2_file_2",
					"image_2_file_3": "path_to_image_2_file_3",
				},
				"image_3": {
					"image_3_file_1": "path_to_image_3_file_1",
					"image_3_file_2": "path_to_image_3_file_2",
					"image_3_file_3": "path_to_image_3_file_3",
				},
			},
			metadataFiles: nil,
			newMockRoundTripper: func() *mockRoundTripper {
				mrt := &mockRoundTripper{}

				mrt.On("RoundTrip", "POST", "dataset/create").Return(&http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     make(http.Header),
				}, nil).Times(3)

				return mrt
			},
			mockRoundTripperAssert: func(t *testing.T, mrt *mockRoundTripper) {
				mrt.AssertExpectations(t)
				assert.Len(t, mrt.requestsSeen, 3)
				assert.ElementsMatch(t, append(mrt.requestsSeen[0].FileAccessionIDs, append(mrt.requestsSeen[1].FileAccessionIDs, mrt.requestsSeen[2].FileAccessionIDs...)...), []string{"image_1_file_1", "image_1_file_2", "image_1_file_3", "image_2_file_1", "image_2_file_2", "image_2_file_3", "image_3_file_1", "image_3_file_2", "image_3_file_3"})
				assert.Equal(t,
					map[string]string{
						"image_1_file_1": "IMAGES/IMAGE_image_1/path_to_image_1_file_1.c4gh",
						"image_1_file_2": "IMAGES/IMAGE_image_1/path_to_image_1_file_2.c4gh",
						"image_1_file_3": "IMAGES/IMAGE_image_1/path_to_image_1_file_3.c4gh",
						"image_2_file_1": "IMAGES/IMAGE_image_2/path_to_image_2_file_1.c4gh",
						"image_2_file_2": "IMAGES/IMAGE_image_2/path_to_image_2_file_2.c4gh",
						"image_2_file_3": "IMAGES/IMAGE_image_2/path_to_image_2_file_3.c4gh",
						"image_3_file_1": "IMAGES/IMAGE_image_3/path_to_image_3_file_1.c4gh",
						"image_3_file_2": "IMAGES/IMAGE_image_3/path_to_image_3_file_2.c4gh",
						"image_3_file_3": "IMAGES/IMAGE_image_3/path_to_image_3_file_3.c4gh",
					},
					mergeMaps(
						mrt.requestsSeen[0].FileDownloadPaths,
						mrt.requestsSeen[1].FileDownloadPaths,
						mrt.requestsSeen[2].FileDownloadPaths,
					),
				)
			},
			datasetCreateFilesBatchSize: 3,
			expectedError:               nil,
		}, {
			name:               "1_call",
			datasetAccessionId: "123",
			imageAccessionFileNames: map[string]map[string]string{
				"image_1": {
					"image_1_file_1": "path_to_image_1_file_1",
					"image_1_file_2": "path_to_image_1_file_2",
					"image_1_file_3": "path_to_image_1_file_3",
				},
				"image_2": {
					"image_2_file_1": "path_to_image_2_file_1",
					"image_2_file_2": "path_to_image_2_file_2",
					"image_2_file_3": "path_to_image_2_file_3",
				},
				"image_3": {
					"image_3_file_1": "path_to_image_3_file_1",
					"image_3_file_2": "path_to_image_3_file_2",
					"image_3_file_3": "path_to_image_3_file_3",
				},
			},
			metadataFiles: map[metadata_models.MetadataFileType]string{
				metadata_models.MetadataFileTypeDataset:     "1",
				metadata_models.MetadataFileTypeImage:       "2",
				metadata_models.MetadataFileTypeObservation: "3",
			},
			newMockRoundTripper: func() *mockRoundTripper {
				mrt := &mockRoundTripper{}

				mrt.On("RoundTrip", "POST", "dataset/create").Return(&http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     make(http.Header),
				}, nil).Once()

				return mrt
			},
			mockRoundTripperAssert: func(t *testing.T, mrt *mockRoundTripper) {
				mrt.AssertExpectations(t)
				assert.Len(t, mrt.requestsSeen, 1)
				assert.ElementsMatch(t, mrt.requestsSeen[0].FileAccessionIDs, []string{"1", "2", "3", "image_1_file_1", "image_1_file_2", "image_1_file_3", "image_2_file_1", "image_2_file_2", "image_2_file_3", "image_3_file_1", "image_3_file_2", "image_3_file_3"})
				assert.Equal(t,
					map[string]string{
						"1":              "METADATA/dataset.xml.c4gh",
						"2":              "METADATA/image.xml.c4gh",
						"3":              "METADATA/observation.xml.c4gh",
						"image_1_file_1": "IMAGES/IMAGE_image_1/path_to_image_1_file_1.c4gh",
						"image_1_file_2": "IMAGES/IMAGE_image_1/path_to_image_1_file_2.c4gh",
						"image_1_file_3": "IMAGES/IMAGE_image_1/path_to_image_1_file_3.c4gh",
						"image_2_file_1": "IMAGES/IMAGE_image_2/path_to_image_2_file_1.c4gh",
						"image_2_file_2": "IMAGES/IMAGE_image_2/path_to_image_2_file_2.c4gh",
						"image_2_file_3": "IMAGES/IMAGE_image_2/path_to_image_2_file_3.c4gh",
						"image_3_file_1": "IMAGES/IMAGE_image_3/path_to_image_3_file_1.c4gh",
						"image_3_file_2": "IMAGES/IMAGE_image_3/path_to_image_3_file_2.c4gh",
						"image_3_file_3": "IMAGES/IMAGE_image_3/path_to_image_3_file_3.c4gh",
					},
					mrt.requestsSeen[0].FileDownloadPaths,
				)
			},
			datasetCreateFilesBatchSize: 15,
			expectedError:               nil,
		}, {
			name:               "many_calls",
			datasetAccessionId: "123",
			imageAccessionFileNames: map[string]map[string]string{
				"image_1": {
					"image_1_file_1": "path_to_image_1_file_1",
					"image_1_file_2": "path_to_image_1_file_2",
					"image_1_file_3": "path_to_image_1_file_3",
				},
				"image_2": {
					"image_2_file_1": "path_to_image_2_file_1",
					"image_2_file_2": "path_to_image_2_file_2",
					"image_2_file_3": "path_to_image_2_file_3",
				},
				"image_3": {
					"image_3_file_1": "path_to_image_3_file_1",
					"image_3_file_2": "path_to_image_3_file_2",
					"image_3_file_3": "path_to_image_3_file_3",
				},
			},
			metadataFiles: map[metadata_models.MetadataFileType]string{
				metadata_models.MetadataFileTypeDataset:     "1",
				metadata_models.MetadataFileTypeImage:       "2",
				metadata_models.MetadataFileTypeObservation: "3",
			},
			newMockRoundTripper: func() *mockRoundTripper {
				mrt := &mockRoundTripper{}

				mrt.On("RoundTrip", "POST", "dataset/create").Return(&http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     make(http.Header),
				}, nil).Times(12)

				return mrt
			},
			mockRoundTripperAssert: func(t *testing.T, mrt *mockRoundTripper) {
				mrt.AssertExpectations(t)
				assert.Len(t, mrt.requestsSeen, 12)

				assert.ElementsMatch(t,
					append(mrt.requestsSeen[0].FileAccessionIDs,
						append(mrt.requestsSeen[1].FileAccessionIDs,
							append(mrt.requestsSeen[2].FileAccessionIDs,
								append(mrt.requestsSeen[3].FileAccessionIDs,
									append(mrt.requestsSeen[4].FileAccessionIDs,
										append(mrt.requestsSeen[5].FileAccessionIDs,
											append(mrt.requestsSeen[6].FileAccessionIDs,
												append(mrt.requestsSeen[7].FileAccessionIDs,
													append(mrt.requestsSeen[8].FileAccessionIDs,
														append(mrt.requestsSeen[9].FileAccessionIDs,
															append(mrt.requestsSeen[10].FileAccessionIDs,
																mrt.requestsSeen[11].FileAccessionIDs...)...)...)...)...)...)...)...)...)...)...),
					[]string{"1", "2", "3", "image_1_file_1", "image_1_file_2", "image_1_file_3", "image_2_file_1", "image_2_file_2", "image_2_file_3", "image_3_file_1", "image_3_file_2", "image_3_file_3"})
				assert.Equal(t,
					map[string]string{
						"1":              "METADATA/dataset.xml.c4gh",
						"2":              "METADATA/image.xml.c4gh",
						"3":              "METADATA/observation.xml.c4gh",
						"image_1_file_1": "IMAGES/IMAGE_image_1/path_to_image_1_file_1.c4gh",
						"image_1_file_2": "IMAGES/IMAGE_image_1/path_to_image_1_file_2.c4gh",
						"image_1_file_3": "IMAGES/IMAGE_image_1/path_to_image_1_file_3.c4gh",
						"image_2_file_1": "IMAGES/IMAGE_image_2/path_to_image_2_file_1.c4gh",
						"image_2_file_2": "IMAGES/IMAGE_image_2/path_to_image_2_file_2.c4gh",
						"image_2_file_3": "IMAGES/IMAGE_image_2/path_to_image_2_file_3.c4gh",
						"image_3_file_1": "IMAGES/IMAGE_image_3/path_to_image_3_file_1.c4gh",
						"image_3_file_2": "IMAGES/IMAGE_image_3/path_to_image_3_file_2.c4gh",
						"image_3_file_3": "IMAGES/IMAGE_image_3/path_to_image_3_file_3.c4gh",
					},
					mergeMaps(
						mrt.requestsSeen[0].FileDownloadPaths,
						mrt.requestsSeen[1].FileDownloadPaths,
						mrt.requestsSeen[2].FileDownloadPaths,
						mrt.requestsSeen[3].FileDownloadPaths,
						mrt.requestsSeen[4].FileDownloadPaths,
						mrt.requestsSeen[5].FileDownloadPaths,
						mrt.requestsSeen[6].FileDownloadPaths,
						mrt.requestsSeen[7].FileDownloadPaths,
						mrt.requestsSeen[8].FileDownloadPaths,
						mrt.requestsSeen[9].FileDownloadPaths,
						mrt.requestsSeen[10].FileDownloadPaths,
						mrt.requestsSeen[11].FileDownloadPaths,
					),
				)
			},
			datasetCreateFilesBatchSize: 1,
			expectedError:               nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mrt := tc.newMockRoundTripper()
			dmfhTester := &dodMetadataFileHandler{
				ctx: nil, // not needed for triggerDatasetCreation test
				httpClient: &http.Client{
					Transport: mrt,
				},
				transferManagerClient: nil,        // not needed for triggerDatasetCreation test
				c4ghPublicKey:         [32]byte{}, // not needed for triggerDatasetCreation test
				uploadUser:            tc.name,
			}
			old := datasetCreateFilesBatchSize
			datasetCreateFilesBatchSize = tc.datasetCreateFilesBatchSize
			t.Cleanup(func() {
				datasetCreateFilesBatchSize = old
			})

			assert.Equal(t, tc.expectedError,
				dmfhTester.triggerDatasetCreation(
					context.Background(),
					tc.datasetAccessionId,
					tc.metadataFiles,
					tc.imageAccessionFileNames,
				),
			)

			tc.mockRoundTripperAssert(t, mrt)
		})
	}
}

func mergeMaps[K comparable, V any](maps ...map[K]V) map[K]V {
	out := make(map[K]V)
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}

	return out
}
