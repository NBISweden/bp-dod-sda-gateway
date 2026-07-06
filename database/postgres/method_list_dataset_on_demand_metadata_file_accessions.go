package postgres

import (
	"context"
	"database/sql"

	"github.com/NBISweden/bp-dod-sda-gateway/models/metadata_models"
)

const listDatasetOnDemandMetadataFileAccessionsQuery = "listDatasetOnDemandMetadataFileAccessions"

func init() {
	queries[listDatasetOnDemandMetadataFileAccessionsQuery] = `SELECT mf.dataset_accession, mf.type, mf.accession
FROM dod_dataset_metadata_file AS mf
INNER JOIN dod_dataset AS dd ON dd.accession = mf.dataset_accession
WHERE dd.released_at IS NULL
`
}
func (db *pgDb) listDatasetOnDemandMetadataFileAccessions(ctx context.Context, tx *sql.Tx) (map[string]map[metadata_models.MetadataFileType]string, error) {
	stmt, err := db.getPreparedStmt(tx, listDatasetOnDemandMetadataFileAccessionsQuery)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	dodMetadataFilesAccession := make(map[string]map[metadata_models.MetadataFileType]string)

	for rows.Next() {

		var datasetAccession, accession string
		var metadataType metadata_models.MetadataFileType

		if err := rows.Scan(&datasetAccession, &metadataType, &accession); err != nil {
			return nil, err
		}

		if datasetMetadataFiles, ok := dodMetadataFilesAccession[datasetAccession]; ok {
			datasetMetadataFiles[metadataType] = accession
		} else {
			dodMetadataFilesAccession[datasetAccession] = make(map[metadata_models.MetadataFileType]string)
			dodMetadataFilesAccession[datasetAccession][metadataType] = accession
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return dodMetadataFilesAccession, nil
}
