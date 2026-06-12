package postgres

import (
	"context"
	"database/sql"
)

const setDatasetOnDemandDatasetReleasedQuery = "setDatasetOnDemandDatasetReleased"

func init() {
	queries[setDatasetOnDemandDatasetReleasedQuery] = `
UPDATE dod_dataset 
SET released_at = clock_timestamp() 
WHERE accession = $1;
`
}
func (db *pgDb) setDatasetOnDemandDatasetReleased(ctx context.Context, tx *sql.Tx, datasetAccession string) error {
	insertDatasetStmt, err := db.getPreparedStmt(tx, setDatasetOnDemandDatasetReleasedQuery)
	if err != nil {
		return err
	}

	if _, err := insertDatasetStmt.ExecContext(ctx,
		datasetAccession,
	); err != nil {
		return err
	}

	return nil
}
