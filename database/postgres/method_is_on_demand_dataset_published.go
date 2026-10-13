package postgres

import (
	"context"
	"database/sql"
)

const isOnDemandDatasetReleasedQuery = "isOnDemandDatasetReleased"

func init() {
	queries[isOnDemandDatasetReleasedQuery] = `SELECT true
FROM on_demand_dataset  
WHERE accession = $1
AND released_at IS NOT NULL;`
}
func (db *pgDb) isOnDemandDatasetReleased(ctx context.Context, tx *sql.Tx, accession string) (bool, error) {
	stmt, err := db.getPreparedStmt(tx, isOnDemandDatasetReleasedQuery)
	if err != nil {
		return false, err
	}

	var released bool

	if err := stmt.QueryRowContext(ctx, accession).Scan(&released); err != nil {
		return false, err
	}

	return released, nil
}
