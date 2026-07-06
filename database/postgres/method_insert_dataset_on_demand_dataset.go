package postgres

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"

	"github.com/imi-bigpicture/bp-dod-sda-gateway/models"
)

const insertDatasetOnDemandDatasetQuery = "insertDatasetOnDemandDataset"
const insertDatasetOnDemandDatasetMetadataFileQuery = "insertDatasetOnDemandDatasetMetadataFile"
const insertDatasetOnDemandDatasetCreatedFromOriginQuery = "insertDatasetOnDemandDatasetCreatedFromOrigin"

func init() {
	queries[insertDatasetOnDemandDatasetQuery] = `INSERT INTO dod_dataset (accession, requested_by_user)
VALUES($1,$2)
`
	queries[insertDatasetOnDemandDatasetMetadataFileQuery] = `INSERT INTO dod_dataset_metadata_file (dataset_accession, type, accession, xml_content)
VALUES($1, $2, $3, $4)
`
	queries[insertDatasetOnDemandDatasetCreatedFromOriginQuery] = `INSERT INTO dod_dataset_created_from_origin (dod_accession, origin_accession)
VALUES($1, $2)
`
}
func (db *pgDb) insertDatasetOnDemandDataset(ctx context.Context, tx *sql.Tx, dodDataset *models.DatasetOnDemandDataset) error {
	insertDatasetStmt, err := db.getPreparedStmt(tx, insertDatasetOnDemandDatasetQuery)
	if err != nil {
		return err
	}
	insertMetadataFileStmt, err := db.getPreparedStmt(tx, insertDatasetOnDemandDatasetMetadataFileQuery)
	if err != nil {
		return err
	}
	insertCreatedFromStmt, err := db.getPreparedStmt(tx, insertDatasetOnDemandDatasetCreatedFromOriginQuery)
	if err != nil {
		return err
	}

	if _, err := insertDatasetStmt.ExecContext(ctx,
		dodDataset.Accession,
		dodDataset.RequestedByUser,
	); err != nil {
		return fmt.Errorf("failed to insert dod dataset: %w", err)
	}

	for _, createdFrom := range dodDataset.OriginDatasetAccessions {
		if _, err := insertCreatedFromStmt.ExecContext(ctx,
			dodDataset.Accession,
			createdFrom,
		); err != nil {
			return fmt.Errorf("failed to insert dod dataset created from origin: %w", err)
		}
	}

	datasetXml, err := xml.Marshal(dodDataset.DatasetMetadata.MetadataSet)
	if err != nil {
		return fmt.Errorf("failed to marshal dataset: %w", err)
	}

	if _, err := insertMetadataFileStmt.ExecContext(ctx,
		dodDataset.Accession,
		dodDataset.DatasetMetadata.MetadataFileType(),
		dodDataset.DatasetMetadata.Accession,
		datasetXml,
	); err != nil {
		return fmt.Errorf("failed to insert dataset metadata: %w", err)
	}

	imageXml, err := xml.Marshal(dodDataset.ImageMetadata.MetadataSet)
	if err != nil {
		return fmt.Errorf("failed to marshal image: %w", err)
	}
	if _, err := insertMetadataFileStmt.ExecContext(ctx,
		dodDataset.Accession,
		dodDataset.ImageMetadata.MetadataFileType(),
		dodDataset.ImageMetadata.Accession,
		imageXml,
	); err != nil {
		return fmt.Errorf("failed to insert image metadata: %w", err)
	}

	observationXml, err := xml.Marshal(dodDataset.ObservationMetadata.MetadataSet)
	if err != nil {
		return fmt.Errorf("failed to marshal observation: %w", err)
	}
	if _, err := insertMetadataFileStmt.ExecContext(ctx,
		dodDataset.Accession,
		dodDataset.ObservationMetadata.MetadataFileType(),
		dodDataset.ObservationMetadata.Accession,
		observationXml,
	); err != nil {
		return fmt.Errorf("failed to insert observation metadata: %w", err)
	}

	if dodDataset.ObserverMetadata != nil {
		observerXml, err := xml.Marshal(dodDataset.ObserverMetadata.MetadataSet)
		if err != nil {
			return fmt.Errorf("failed to marshal observer: %w", err)
		}
		if _, err := insertMetadataFileStmt.ExecContext(ctx,
			dodDataset.Accession,
			dodDataset.ObserverMetadata.MetadataFileType(),
			dodDataset.ObserverMetadata.Accession,
			observerXml,
		); err != nil {
			return fmt.Errorf("failed to insert observer metadata: %w", err)
		}
	}

	policyXml, err := xml.Marshal(dodDataset.PolicyMetadata.MetadataSet)
	if err != nil {
		return fmt.Errorf("failed to marshal policy: %w", err)
	}
	if _, err := insertMetadataFileStmt.ExecContext(ctx,
		dodDataset.Accession,
		dodDataset.PolicyMetadata.MetadataFileType(),
		dodDataset.PolicyMetadata.Accession,
		policyXml,
	); err != nil {
		return fmt.Errorf("failed to insert policy metadata: %w", err)
	}

	remsXml, err := xml.Marshal(dodDataset.RemsMetadata.MetadataSet)
	if err != nil {
		return fmt.Errorf("failed to marshal policy: %w", err)
	}
	if _, err := insertMetadataFileStmt.ExecContext(ctx,
		dodDataset.Accession,
		dodDataset.RemsMetadata.MetadataFileType(),
		dodDataset.RemsMetadata.Accession,
		remsXml,
	); err != nil {
		return fmt.Errorf("failed to insert rems metadata: %w", err)
	}

	sampleXml, err := xml.Marshal(dodDataset.SampleMetadata.MetadataSet)
	if err != nil {
		return fmt.Errorf("failed to marshal sample: %w", err)
	}
	if _, err := insertMetadataFileStmt.ExecContext(ctx,
		dodDataset.Accession,
		dodDataset.SampleMetadata.MetadataFileType(),
		dodDataset.SampleMetadata.Accession,
		sampleXml,
	); err != nil {
		return fmt.Errorf("failed to insert sample metadata: %w", err)
	}

	stainingXml, err := xml.Marshal(dodDataset.StainingMetadata.MetadataSet)
	if err != nil {
		return fmt.Errorf("failed to marshal staining: %w", err)
	}
	if _, err := insertMetadataFileStmt.ExecContext(ctx,
		dodDataset.Accession,
		dodDataset.StainingMetadata.MetadataFileType(),
		dodDataset.StainingMetadata.Accession,
		stainingXml,
	); err != nil {
		return fmt.Errorf("failed to insert staining metadata: %w", err)
	}

	return nil
}
