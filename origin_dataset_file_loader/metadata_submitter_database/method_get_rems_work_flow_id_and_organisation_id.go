package metadata_submitter_database

import (
	"context"
	"encoding/xml"
	"fmt"
	"strconv"

	"github.com/NBISweden/bp-dod-sda-gateway/models/metadata_models"
)

const getRemsWorkFlowIDAndOrganisationIDQuery = "getRemsWorkFlowIDAndOrganisationID"

func init() {
	queries[getRemsWorkFlowIDAndOrganisationIDQuery] = `
SELECT xml_document
FROM objects
WHERE submission_id = $1
AND object_type = 'rems';
`
}
func (db *metadataSubmitterPg) getRemsWorkFlowIDAndOrganisationID(ctx context.Context, datasetAccession string) (int, string, error) {
	stmt, err := db.getPreparedStmt(getRemsWorkFlowIDAndOrganisationIDQuery)
	if err != nil {
		return -1, "", err
	}

	row := stmt.QueryRowContext(ctx,
		datasetAccession,
	)
	if err := row.Err(); err != nil {
		return -1, "", err
	}

	var remsEntryXmlContent []byte
	if err := row.Scan(&remsEntryXmlContent); err != nil {
		return -1, "", err
	}

	var remsEntry metadata_models.Rems
	if err := xml.Unmarshal(remsEntryXmlContent, &remsEntry); err != nil {
		return -1, "", fmt.Errorf("failed to unmarshal rems: %w", err)
	}
	workflowID, err := strconv.Atoi(remsEntry.WorkflowId)
	if err != nil {
		return -1, "", fmt.Errorf("failed to parse workflow id to an integer: %w", err)
	}

	return workflowID, remsEntry.OrganisationId, nil
}
