package postgres

import (
	"context"
	"database/sql"
)

const insertOnDemandDatasetImageQuery = "insertOnDemandDatasetImage"

func init() {
	queries[insertOnDemandDatasetImageQuery] = `INSERT INTO on_demand_dataset_image (dod_accession, origin_accession, image_accession)
VALUES($1, $2, $3)
`
}
func (db *pgDb) insertOnDemandDatasetImage(ctx context.Context, tx *sql.Tx, dodAccession, originAccession, imageAccession string) error {
	stmt, err := db.getPreparedStmt(tx, insertOnDemandDatasetImageQuery)
	if err != nil {
		return err
	}

	if _, err := stmt.ExecContext(ctx,
		dodAccession, originAccession, imageAccession,
	); err != nil {
		return err
	}

	return nil
}
