package postgres

import (
	"context"
	"database/sql"
)

const listDatasetOnDemandImageFileAccessionsQuery = "listDatasetOnDemandImageFileAccessions"

func init() {
	queries[listDatasetOnDemandImageFileAccessionsQuery] = `
SELECT if.accession
FROM dod_image as dodi
INNER JOIN dataset_image AS di ON di.alias = dodi.image_alias AND di.dataset_accession = dodi.origin_accession
INNER JOIN image_file AS if ON if.image_alias = di.alias AND if.dataset_accession = di.dataset_accession
WHERE dodi.dod_accession = $1
`
}
func (db *pgDb) listDatasetOnDemandImageFileAccessions(ctx context.Context, tx *sql.Tx, dataseAccession string) ([]string, error) {
	stmt, err := db.getPreparedStmt(tx, listDatasetOnDemandImageFileAccessionsQuery)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, dataseAccession)
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
