package postgres

import (
	"context"
	"database/sql"
)

const listUserEmailsByDatasetAccessionQuery = "listUserEmailsByDatasetAccession"

func init() {
	queries[listUserEmailsByDatasetAccessionQuery] = `SELECT user_email
FROM on_demand_dataset_request 
WHERE on_demand_dataset_accession = $1
`
}

func (db *pgDb) listUserEmailsByDatasetAccession(ctx context.Context, tx *sql.Tx, datasetAccession string) ([]string, error) {
	stmt, err := db.getPreparedStmt(tx, listUserEmailsByDatasetAccessionQuery)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, datasetAccession)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var userEmails []string

	for rows.Next() {
		var email string

		if err := rows.Scan(&email); err != nil {
			return nil, err
		}

		userEmails = append(userEmails, email)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return userEmails, nil
}
