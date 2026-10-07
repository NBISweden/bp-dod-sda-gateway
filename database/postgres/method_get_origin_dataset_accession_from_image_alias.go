package postgres

import (
	"context"
	"database/sql"
	"errors"
)

const getOriginDatasetAccessionFromImageAccessionQuery = "getOriginDatasetAccessionFromImageAccession"

func init() {
	queries[getOriginDatasetAccessionFromImageAccessionQuery] = `SELECT dataset_accession
FROM dataset_image
WHERE accession = $1;`
}
func (db *pgDb) getOriginDatasetAccessionFromImageAccession(ctx context.Context, tx *sql.Tx, imageAccession string) (string, error) {
	stmt, err := db.getPreparedStmt(tx, getOriginDatasetAccessionFromImageAccessionQuery)
	if err != nil {
		return "", err
	}
	var accession string

	if err := stmt.QueryRowContext(ctx, imageAccession).Scan(&accession); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}

		return "", err
	}

	return accession, nil
}
