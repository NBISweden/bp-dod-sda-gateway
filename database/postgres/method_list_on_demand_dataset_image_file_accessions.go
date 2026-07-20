package postgres

import (
	"context"
	"database/sql"
)

const listDatasetOnDemandImageFileAccessionsQuery = "listDatasetOnDemandImageFileAccessions"

func init() {
	queries[listDatasetOnDemandImageFileAccessionsQuery] = `SELECT if.accession
FROM on_demand_dataset_image as oddi
INNER JOIN dataset_image AS di ON di.accession = oddi.image_accession
INNER JOIN image_file AS if ON if.image_accession = di.accession
WHERE oddi.dod_accession = $1
`
}
func (db *pgDb) listDatasetOnDemandImageFileAccessions(ctx context.Context, tx *sql.Tx, datasetAccession string) ([]string, error) {
	stmt, err := db.getPreparedStmt(tx, listDatasetOnDemandImageFileAccessionsQuery)
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

	var imageFileAccessions []string

	for rows.Next() {
		var accession string

		if err := rows.Scan(&accession); err != nil {
			return nil, err
		}
		imageFileAccessions = append(imageFileAccessions, accession)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return imageFileAccessions, nil
}
