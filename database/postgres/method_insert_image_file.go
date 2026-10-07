package postgres

import (
	"context"
	"database/sql"
)

const insertImageFileQuery = "insertImageFile"

func init() {
	queries[insertImageFileQuery] = `INSERT INTO image_file (dataset_accession, image_accession, accession, base_file_name)
VALUES($1, $2, $3, $4);`
}
func (db *pgDb) insertImageFile(ctx context.Context, tx *sql.Tx, datasetAccession, imageAccession, fileAccession, baseFileName string) error {
	stmt, err := db.getPreparedStmt(tx, insertImageFileQuery)
	if err != nil {
		return err
	}

	result, err := stmt.ExecContext(ctx,
		datasetAccession,
		imageAccession,
		fileAccession,
		baseFileName,
	)
	if err != nil {
		return err
	}
	if rowsAffected, _ := result.RowsAffected(); rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
