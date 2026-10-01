package metadata_submitter

import (
	"context"
)

func (mdc *metadataSubmitterClient) GetRemsWorkFlowIDAndOrganisationID(ctx context.Context, datasetAccession string) (int, string, error) {
	entry, err := mdc.getDatasetMetadata(ctx, datasetAccession)
	if err != nil {
		return 0, "", err
	}

	return entry.originDataset.RemsWorkflowID, entry.originDataset.RemsOrganisationID, nil
}
