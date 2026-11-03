package postgres

import (
	"context"
	"database/sql"
	"errors"
)

const GetOnDemandDatasetFromImageAccessionsHashQuery = "GetOnDemandDatasetFromImageAccessionsHash"

func init() {
	queries[GetOnDemandDatasetFromImageAccessionsHashQuery] = `SELECT accession, released_at IS NOT NULL AS released 
FROM on_demand_dataset  
WHERE image_accessions_hash = $1;`
}

func (db *pgDb) getOnDemandDatasetFromImageAccessionsHash(ctx context.Context, tx *sql.Tx, imageAccessionHash string) (string, bool, error) {
	stmt, err := db.getPreparedStmt(tx, GetOnDemandDatasetFromImageAccessionsHashQuery)
	if err != nil {
		return "", false, err
	}

	var accession string
	var released bool

	if err := stmt.QueryRowContext(ctx, imageAccessionHash).Scan(&accession, &released); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}

		return "", false, err
	}

	return accession, released, nil
}
