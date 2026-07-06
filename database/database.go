package database

import (
	"context"

	"github.com/NBISweden/bp-dod-sda-gateway/models"
	"github.com/NBISweden/bp-dod-sda-gateway/models/metadata_models"
)

type Transaction interface {
	// Commit the transaction
	Commit() error
	// Rollback the transaction
	Rollback() error
	functions
}

type Database interface {
	// BeginTransaction starts a database transaction, either commit or rollback needs to be called when done to release resources and close transaction
	BeginTransaction(ctx context.Context) (Transaction, error)
	// Close the database connection
	Close() error
	SchemaVersion() (uint, error)
	Ping(ctx context.Context) error

	functions
}

// functions denotes the available database functions
type functions interface {
	GetOriginDataset(ctx context.Context, accession string) (*models.OriginDataset, error)
	GetOriginDatasetAccessionFromImageAlias(ctx context.Context, imageAlias string) (string, error)
	InsertOriginDataset(ctx context.Context, originDataset *models.OriginDataset) error

	InsertDatasetOnDemandDataset(ctx context.Context, dodDataset *models.DatasetOnDemandDataset) error
	InsertDatasetOnDemandDatasetImage(ctx context.Context, dodAccession, originAccession, imageAlias string) error

	InsertDatasetImage(ctx context.Context, datasetAccession, imageAlias string) error
	InsertImageFile(ctx context.Context, datasetAccession, imageAlias, fileAlias string) error

	ListDatasetOnDemandMetadataFiles(ctx context.Context) (map[string]map[metadata_models.MetadataFileType]string, error)

	ListDatasetOnDemandImageFileAccessions(ctx context.Context, datasetAccession string) ([]string, error)

	GetDatasetOnDemandDatasetRemsMetadata(ctx context.Context, datasetAccession string) (*metadata_models.RemsSet, error)
	IsDatasetOnDemandDatasetReleased(ctx context.Context, dodDatasetAccession string) (bool, error)

	SetDatasetOnDemandDatasetReleased(ctx context.Context, datasetAccession string) error
}

var db Database

// RegisterDatabase registers the database implementation to be used
func RegisterDatabase(d Database) {
	db = d
}

func Close() error {
	return db.Close()
}

// BeginTransaction starts a database transaction, either commit or rollback needs to be called when done to release resources and close transaction
func BeginTransaction(ctx context.Context) (Transaction, error) {
	return db.BeginTransaction(ctx)
}

func ListDatasetOnDemandMetadataFiles(ctx context.Context) (map[string]map[metadata_models.MetadataFileType]string, error) {
	return db.ListDatasetOnDemandMetadataFiles(ctx)
}

func ListDatasetOnDemandImageFileAccessions(ctx context.Context, datasetAccession string) ([]string, error) {
	return db.ListDatasetOnDemandImageFileAccessions(ctx, datasetAccession)
}

func SetDatasetOnDemandDatasetReleased(ctx context.Context, datasetAccession string) error {
	return db.SetDatasetOnDemandDatasetReleased(ctx, datasetAccession)
}

func GetDatasetOnDemandDatasetRemsMetadata(ctx context.Context, datasetAccession string) (*metadata_models.RemsSet, error) {
	return db.GetDatasetOnDemandDatasetRemsMetadata(ctx, datasetAccession)
}

func IsDatasetOnDemandDatasetPublished(ctx context.Context, dodDatasetAccession string) (bool, error) {
	return db.IsDatasetOnDemandDatasetReleased(ctx, dodDatasetAccession)
}
