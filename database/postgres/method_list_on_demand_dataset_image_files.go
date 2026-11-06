package postgres

import (
	"context"
	"database/sql"
)

const listOnDemandImageFilesQuery = "listOnDemandImageFiles"

func init() {
	queries[listOnDemandImageFilesQuery] = `SELECT  if.accession, if.image_accession, if.base_file_name
FROM on_demand_dataset_image as oddi
INNER JOIN dataset_image AS di ON di.accession = oddi.image_accession
INNER JOIN image_file AS if ON if.image_accession = di.accession
WHERE oddi.on_demand_dataset_accession = $1
`
}
func (db *pgDb) listOnDemandDatasetImageFiles(ctx context.Context, tx *sql.Tx, onDemandDatasetAccession string) (map[string]map[string]string, error) {
	stmt, err := db.getPreparedStmt(tx, listOnDemandImageFilesQuery)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, onDemandDatasetAccession)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	imageAccessionImageFileNames := make(map[string]map[string]string)

	for rows.Next() {
		var accession, imageAccession, baseFileName string

		if err := rows.Scan(&accession, &imageAccession, &baseFileName); err != nil {
			return nil, err
		}

		imageFilesNames, ok := imageAccessionImageFileNames[imageAccession]
		if !ok {
			imageFilesNames = make(map[string]string)
		}
		imageFilesNames[accession] = baseFileName
		imageAccessionImageFileNames[imageAccession] = imageFilesNames
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return imageAccessionImageFileNames, nil
}
