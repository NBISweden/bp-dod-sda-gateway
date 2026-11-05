package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/NBISweden/bp-dod-sda-gateway/database"
	"github.com/lib/pq"
)

const insertOnDemandDatasetRequestQuery = "insertOnDemandDatasetRequest"

func init() {
	queries[insertOnDemandDatasetRequestQuery] = `INSERT INTO on_demand_dataset_request (requested_by_user, user_email, on_demand_dataset_accession, label)
VALUES($1, $2, $3, $4);`
}
func (db *pgDb) insertOnDemandDatasetRequest(ctx context.Context, tx *sql.Tx, user, email, datasetAccession, label string) error {
	stmt, err := db.getPreparedStmt(tx, insertOnDemandDatasetRequestQuery)
	if err != nil {
		return err
	}

	if _, err := stmt.ExecContext(ctx,
		user,
		email,
		datasetAccession,
		label,
	); err != nil {
		if pqErr, ok := errors.AsType[*pq.Error](err); ok && pqErr.Code == "23505" {
			return database.ErrAlreadyExists
		}

		return fmt.Errorf("failed to insert dod dataset: %w", err)
	}

	return nil
}
