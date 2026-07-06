package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/NBISweden/bp-dod-sda-gateway/internal/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const getOriginDatasetAccessionFromImageAliasQuery = "getOriginDatasetAccessionFromImageAlias"

func init() {
	queries[getOriginDatasetAccessionFromImageAliasQuery] = `SELECT od.accession
FROM origin_dataset AS od
INNER JOIN dataset_image AS di ON di.dataset_accession = od.accession
WHERE di.alias = $1

`
}
func (db *pgDb) getOriginDatasetAccessionFromImageAlias(ctx context.Context, tx *sql.Tx, imageAlias string) (string, error) {
	ctx, span := observability.Tracer().Start(ctx, "getOriginDatasetAccessionFromImageAlias", trace.WithAttributes(attribute.String("image-alias", imageAlias)))
	defer span.End()

	stmt, err := db.getPreparedStmt(tx, getOriginDatasetAccessionFromImageAliasQuery)
	if err != nil {
		return "", err
	}
	var accession string

	if err := stmt.QueryRowContext(ctx, imageAlias).Scan(&accession); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}

		return "", err
	}

	return accession, nil
}
