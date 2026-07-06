package postgres

import (
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"

	"github.com/imi-bigpicture/bp-dod-sda-gateway/internal/observability"
	"github.com/imi-bigpicture/bp-dod-sda-gateway/models/metadata_models"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const getDatasetOnDemandDatasetRemsMetadataQuery = "getDatasetOnDemandDatasetRemsMetadata"

func init() {
	queries[getDatasetOnDemandDatasetRemsMetadataQuery] = `SELECT md_rems.xml_content
FROM dod_dataset AS ddd
INNER JOIN dod_dataset_metadata_file AS md_rems ON md_rems.type = 'rems' AND ddd.accession = md_rems.dataset_accession 
WHERE ddd.accession = $1

`
}
func (db *pgDb) getDatasetOnDemandDatasetRemsMetadata(ctx context.Context, tx *sql.Tx, accession string) (*metadata_models.RemsSet, error) {
	ctx, span := observability.Tracer().Start(ctx, "getDatasetOnDemandDatasetRemsMetadata", trace.WithAttributes(attribute.String("accession", accession)))
	defer span.End()

	stmt, err := db.getPreparedStmt(tx, getDatasetOnDemandDatasetRemsMetadataQuery)
	if err != nil {
		return nil, err
	}

	var remsXml []byte

	if err := stmt.QueryRowContext(ctx, accession).Scan(&remsXml); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	remsMetdata := new(metadata_models.RemsSet)
	if err := xml.Unmarshal(remsXml, &remsMetdata); err != nil {
		return nil, fmt.Errorf("failed to marshal rems: %w", err)
	}

	return remsMetdata, nil
}
