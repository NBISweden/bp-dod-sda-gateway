package postgres

import (
	"context"
	"database/sql"
)

const setOnDemandDatasetReleasedQuery = "setOnDemandDatasetReleased"

func init() {
	queries[setOnDemandDatasetReleasedQuery] = `UPDATE on_demand_dataset 
SET released_at = clock_timestamp() 
WHERE accession = $1;
`
}
func (db *pgDb) setOnDemandDatasetReleased(ctx context.Context, tx *sql.Tx, datasetAccession string) error {
	insertDatasetStmt, err := db.getPreparedStmt(tx, setOnDemandDatasetReleasedQuery)
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
