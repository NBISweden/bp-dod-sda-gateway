package postgres

import (
	"context"
	"database/sql"

	"github.com/imi-bigpicture/bp-dod-sda-gateway/internal/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const isDatasetOnDemandDatasetReleasedQuery = "isDatasetOnDemandDatasetReleased"

func init() {
	queries[isDatasetOnDemandDatasetReleasedQuery] = `
SELECT EXISTS(
SELECT 1
FROM dod_dataset  
WHERE accession = $1
AND released_at IS NOT NULL
)
`
}
func (db *pgDb) isDatasetOnDemandDatasetReleased(ctx context.Context, tx *sql.Tx, accession string) (bool, error) {
	ctx, span := observability.Tracer().Start(ctx, "isDatasetOnDemandDatasetReleased", trace.WithAttributes(attribute.String("accession", accession)))
	defer span.End()

	stmt, err := db.getPreparedStmt(tx, isDatasetOnDemandDatasetReleasedQuery)
	if err != nil {
		return false, err
	}

	var released bool

	if err := stmt.QueryRowContext(ctx, accession).Scan(&released); err != nil {
		return false, err
	}

	return released, nil
}
