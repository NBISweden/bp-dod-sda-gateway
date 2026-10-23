package postgres

import (
	"context"
	"database/sql"

	"github.com/NBISweden/bp-dod-sda-gateway/models"
)

const listUsersOnDemandDatasetRequestsQuery = "listUsersOnDemandDatasetRequests"

func init() {
	queries[listUsersOnDemandDatasetRequestsQuery] = `SELECT odd.accession, odd.released_at IS NOT NULL, oddr.requested_at, oddr.label
FROM on_demand_dataset_request as oddr
INNER JOIN on_demand_dataset AS odd ON odd.accession = oddr.on_demand_dataset_accession
WHERE oddr.requested_by = $1
`
}

func (db *pgDb) listUsersOnDemandDatasetRequests(ctx context.Context, tx *sql.Tx, user string) ([]*models.OnDemandDatasetRequest, error) {
	stmt, err := db.getPreparedStmt(tx, listUsersOnDemandDatasetRequestsQuery)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, user)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var usersOnDemandDatasetRequests []*models.OnDemandDatasetRequest

	for rows.Next() {
		oddr := new(models.OnDemandDatasetRequest)

		if err := rows.Scan(&oddr.OnDemandDatasetAccession, &oddr.Released, &oddr.RequestedAt, &oddr.Label); err != nil {
			return nil, err
		}

		usersOnDemandDatasetRequests = append(usersOnDemandDatasetRequests, oddr)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return usersOnDemandDatasetRequests, nil
}
