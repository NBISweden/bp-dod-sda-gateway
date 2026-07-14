package metadata_submitter_database

import (
	"context"
	"strings"

	"github.com/NBISweden/bp-dod-sda-gateway/models/metadata_models"
	"github.com/NBISweden/bp-dod-sda-gateway/origin_dataset_file_loader"
)

const listDatasetFilesQuery = "listDatasetFiles"

func init() {
	queries[listDatasetFilesQuery] = `
SELECT file_id, path, submission_id FROM files WHERE submission_id = $1;
`
}

func metadataTypeFromPath(path string) metadata_models.MetadataFileType {
	switch {
	case strings.HasSuffix(path, "METADATA/dataset.xml.c4gh"):
		return metadata_models.MetadataFileTypeDataset
	case strings.HasSuffix(path, "METADATA/image.xml.c4gh"):
		return metadata_models.MetadataFileTypeImage
	case strings.HasSuffix(path, "METADATA/annotation.xml.c4gh"):
		return metadata_models.MetadataFileTypeAnnotation
	case strings.HasSuffix(path, "METADATA/observation.xml.c4gh"):
		return metadata_models.MetadataFileTypeObservation
	case strings.HasSuffix(path, "METADATA/observer.xml.c4gh"):
		return metadata_models.MetadataFileTypeObserver
	case strings.HasSuffix(path, "METADATA/policy.xml.c4gh"):
		return metadata_models.MetadataFileTypePolicy
	case strings.HasSuffix(path, "METADATA/sample.xml.c4gh"):
		return metadata_models.MetadataFileTypeSample
	case strings.HasSuffix(path, "METADATA/staining.xml.c4gh"):
		return metadata_models.MetadataFileTypeStaining
	default:
		return metadata_models.MetadataFileTypeInvalid
	}
}

func (db *metadataSubmitterPg) listDatasetFiles(ctx context.Context, datasetAccession string) ([]*origin_dataset_file_loader.FileInfo, error) {
	stmt, err := db.getPreparedStmt(listDatasetFilesQuery)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx,
		datasetAccession,
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var files []*origin_dataset_file_loader.FileInfo

	for rows.Next() {
		f := new(origin_dataset_file_loader.FileInfo)

		if err := rows.Scan(&f.Accession, &f.Path, &f.DatasetAccession); err != nil {
			return nil, err
		}

		f.MetadataFileType = metadataTypeFromPath(f.Path)

		files = append(files, f)

	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return files, nil
}
