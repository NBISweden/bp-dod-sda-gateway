package postgres

import (
	"context"
	"database/sql"
)

const insertDatasetOnDemandDatasetImageQuery = "insertDatasetOnDemandDatasetImage"

func init() {
	queries[insertDatasetOnDemandDatasetImageQuery] = `
INSERT INTO dod_image (dod_accession, origin_accession, image_alias)
VALUES($1, $2, $3)
`
}
func (db *pgDb) insertDatasetOnDemandDatasetImage(ctx context.Context, tx *sql.Tx, dodAccession, originAccession, imageAlias string) error {
	stmt, err := db.getPreparedStmt(tx, insertDatasetOnDemandDatasetImageQuery)
	if err != nil {
		return err
	}

	if _, err := stmt.ExecContext(ctx,
		dodAccession, originAccession, imageAlias,
	); err != nil {
		return err
	}

	return nil
}
