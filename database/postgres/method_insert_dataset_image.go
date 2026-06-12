package postgres

import (
	"context"
	"database/sql"
)

const insertDatasetImageQuery = "insertDatasetImage"

func init() {
	queries[insertDatasetImageQuery] = `
INSERT INTO dataset_image (dataset_accession, alias)
VALUES($1, $2)

`
}
func (db *pgDb) insertDatasetImage(ctx context.Context, tx *sql.Tx, datasetAccession, imageAlias string) error {
	stmt, err := db.getPreparedStmt(tx, insertDatasetImageQuery)
	if err != nil {
		return err
	}

	_, err = stmt.ExecContext(ctx,
		datasetAccession,
		imageAlias,
	)
	if err != nil {
		return err
	}

	return nil
}
