package origin_dataset_file_loader

import (
	"context"

	"github.com/imi-bigpicture/bp-dod-sda-gateway/models/metadata_models"
)

type OriginDatasetFileLoader interface {
	UnmarshalFileToXml(ctx context.Context, file *FileInfo, dst any) error
	ListDatasetFiles(ctx context.Context, datasetAccession string) ([]*FileInfo, error)
	GetRemsWorkFlowIDAndOrganisationID(ctx context.Context, datasetAccession string) (int, string, error)
	Ping(ctx context.Context) error
	Close() error
}

type FileInfo struct {
	Accession        string
	Path             string
	DatasetAccession string
	MetadataFileType metadata_models.MetadataFileType
}
