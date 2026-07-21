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
	GetOriginDatasetAccessionFromImageAccession(ctx context.Context, imageAccession string) (string, error)
	InsertOriginDataset(ctx context.Context, originDataset *models.OriginDataset) error

	InsertOnDemandDataset(ctx context.Context, onDemandDataset *models.OnDemandDataset) error
	InsertOnDemandDatasetImage(ctx context.Context, onDemandDatasetAccession, imageAccession string) error

	InsertDatasetImage(ctx context.Context, datasetAccession, imageAccession string) error
	InsertImageFile(ctx context.Context, datasetAccession, imageAccession, fileAccession, baseFileName string) error

	ListUnreleasedOnDemandDatasetMetadataFiles(ctx context.Context) (map[string]map[metadata_models.MetadataFileType]string, error)

	ListOnDemandDatasetImageFiles(ctx context.Context, datasetAccession string) (map[string]string, error)

	GetOnDemandDatasetRemsMetadata(ctx context.Context, datasetAccession string) (*metadata_models.RemsSet, error)
	IsOnDemandDatasetReleased(ctx context.Context, dodDatasetAccession string) (bool, error)

	SetOnDemandDatasetReleased(ctx context.Context, datasetAccession string) error
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

func ListUnreleasedOnDemandDatasetMetadataFiles(ctx context.Context) (map[string]map[metadata_models.MetadataFileType]string, error) {
	return db.ListUnreleasedOnDemandDatasetMetadataFiles(ctx)
}

func ListOnDemandDatasetImageFileAccessions(ctx context.Context, datasetAccession string) (map[string]string, error) {
	return db.ListOnDemandDatasetImageFiles(ctx, datasetAccession)
}

func SetOnDemandDatasetReleased(ctx context.Context, datasetAccession string) error {
	return db.SetOnDemandDatasetReleased(ctx, datasetAccession)
}

func GetOnDemandDatasetRemsMetadata(ctx context.Context, datasetAccession string) (*metadata_models.RemsSet, error) {
	return db.GetOnDemandDatasetRemsMetadata(ctx, datasetAccession)
}

func IsOnDemandDatasetPublished(ctx context.Context, dodDatasetAccession string) (bool, error) {
	return db.IsOnDemandDatasetReleased(ctx, dodDatasetAccession)
}
