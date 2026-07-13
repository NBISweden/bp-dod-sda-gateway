package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/NBISweden/bp-dod-sda-gateway/models"
	"github.com/NBISweden/bp-dod-sda-gateway/models/metadata_models"
)

type pgTx struct {
	tx *sql.Tx
	*pgDb
}

func (tx *pgTx) Commit() error {
	return tx.tx.Commit()
}

func (tx *pgTx) Rollback() error {
	err := tx.tx.Rollback()
	if errors.Is(err, sql.ErrTxDone) {
		return nil
	}

	return err
}

func (tx *pgTx) InsertOriginDataset(ctx context.Context, originDataset *models.OriginDataset) error {
	return tx.insertOriginDataset(ctx, tx.tx, originDataset)
}

func (tx *pgTx) InsertDatasetImage(ctx context.Context, datasetAccession, imageAccession string) error {
	return tx.insertDatasetImage(ctx, tx.tx, datasetAccession, imageAccession)
}

func (tx *pgTx) InsertImageFile(ctx context.Context, datasetAccession, imageAccession, fileAccession string) error {
	return tx.insertImageFile(ctx, tx.tx, datasetAccession, imageAccession, fileAccession)
}

func (tx *pgTx) GetOriginDatasetAccessionFromImageAccession(ctx context.Context, imageAccession string) (string, error) {
	return tx.getOriginDatasetAccessionFromImageAccession(ctx, tx.tx, imageAccession)
}

func (tx *pgTx) GetOriginDataset(ctx context.Context, accession string) (*models.OriginDataset, error) {
	return tx.getOriginDataset(ctx, tx.tx, accession)
}

func (tx *pgTx) InsertDatasetOnDemandDataset(ctx context.Context, dodDataset *models.DatasetOnDemandDataset) error {
	return tx.insertDatasetOnDemandDataset(ctx, tx.tx, dodDataset)
}

func (tx *pgTx) InsertDatasetOnDemandDatasetImage(ctx context.Context, dodAccession, originAccession, imageAccession string) error {
	return tx.insertDatasetOnDemandDatasetImage(ctx, tx.tx, dodAccession, originAccession, imageAccession)
}
func (tx *pgTx) ListDatasetOnDemandMetadataFiles(ctx context.Context) (map[string]map[metadata_models.MetadataFileType]string, error) {
	return tx.listDatasetOnDemandMetadataFileAccessions(ctx, nil)
}

func (tx *pgTx) ListDatasetOnDemandImageFileAccessions(ctx context.Context, datasetAccession string) ([]string, error) {
	return tx.listDatasetOnDemandImageFileAccessions(ctx, tx.tx, datasetAccession)
}

func (tx *pgTx) SetDatasetOnDemandDatasetReleased(ctx context.Context, datasetAccession string) error {
	return tx.setDatasetOnDemandDatasetReleased(ctx, tx.tx, datasetAccession)
}

func (tx *pgTx) IsDatasetOnDemandDatasetReleased(ctx context.Context, dodDatasetAccession string) (bool, error) {
	return tx.isDatasetOnDemandDatasetReleased(ctx, tx.tx, dodDatasetAccession)
}
func (tx *pgTx) GetDatasetOnDemandDatasetRemsMetadata(ctx context.Context, datasetAccession string) (*metadata_models.RemsSet, error) {
	return tx.getDatasetOnDemandDatasetRemsMetadata(ctx, tx.tx, datasetAccession)
}
