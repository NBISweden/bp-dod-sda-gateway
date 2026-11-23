package postgres

import (
	"context"
	"database/sql"
)

const upsertImageFileQuery = "upsertImageFile"

func init() {
	queries[upsertImageFileQuery] = `INSERT INTO image_file (dataset_accession, image_accession, accession, base_file_name)
VALUES($1, $2, $3, $4)
ON CONFLICT DO NOTHING;`
}
func (db *pgDb) upsertImageFile(ctx context.Context, tx *sql.Tx, datasetAccession, imageAccession, fileAccession, baseFileName string) error {
	stmt, err := db.getPreparedStmt(tx, upsertImageFileQuery)
	if err != nil {
		return err
	}

	_, err = stmt.ExecContext(ctx,
		datasetAccession,
		imageAccession,
		fileAccession,
		baseFileName,
	)
	if err != nil {
		return err
	}

	return nil
}
