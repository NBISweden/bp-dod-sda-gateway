package postgres

import (
	"context"
	"database/sql"
	"errors"
)

const getOnDemandDatasetAccessionFromImageAccessionsQuery = "getOnDemandDatasetAccessionFromImageAccessions"

func init() {
	queries[getOnDemandDatasetAccessionFromImageAccessionsQuery] = `SELECT accession
FROM on_demand_dataset  
WHERE image_accessions_hash = $1;`
}

func (db *pgDb) getOnDemandDatasetAccessionFromImageAccessions(ctx context.Context, tx *sql.Tx, imageAccessionHash string) (string, error) {
	stmt, err := db.getPreparedStmt(tx, getOnDemandDatasetAccessionFromImageAccessionsQuery)
	if err != nil {
		return "", err
	}

	var accession string

	if err := stmt.QueryRowContext(ctx, imageAccessionHash).Scan(&accession); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}

		return "", err
	}

	return accession, nil
}
