package postgres

import (
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"

	"github.com/NBISweden/bp-dod-sda-gateway/models/metadata_models"
)

const getOnDemandDatasetRemsMetadataQuery = "getOnDemandDatasetRemsMetadata"

func init() {
	queries[getOnDemandDatasetRemsMetadataQuery] = `SELECT md_rems.xml_content
FROM on_demand_dataset AS odd
INNER JOIN on_demand_dataset_metadata_file AS md_rems ON md_rems.type = 'rems' AND odd.accession = md_rems.dataset_accession 
WHERE odd.accession = $1

`
}
func (db *pgDb) getOnDemandDatasetRemsMetadata(ctx context.Context, tx *sql.Tx, accession string) (*metadata_models.RemsSet, error) {
	stmt, err := db.getPreparedStmt(tx, getOnDemandDatasetRemsMetadataQuery)
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
		return nil, fmt.Errorf("failed to unmarshal rems: %w", err)
	}

	return remsMetdata, nil
}
