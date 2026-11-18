package postgres

import (
	"context"
	"database/sql"
)

const upsertDatasetImageQuery = "upsertDatasetImage"

func init() {
	queries[upsertDatasetImageQuery] = `INSERT INTO dataset_image (dataset_accession, accession)
VALUES($1, $2)
ON CONFLICT (accession) DO UPDATE
SET removed_from_dataset = false;`
}
func (db *pgDb) upsertDatasetImage(ctx context.Context, tx *sql.Tx, datasetAccession, imageAccession string) error {
	stmt, err := db.getPreparedStmt(tx, upsertDatasetImageQuery)
	if err != nil {
		return err
	}

	_, err = stmt.ExecContext(ctx,
		datasetAccession,
		imageAccession,
	)
	if err != nil {
		return err
	}

	return nil
}
