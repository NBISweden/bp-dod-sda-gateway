package postgres

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"

	"github.com/NBISweden/bp-dod-sda-gateway/models"
)

const insertOnDemandDatasetQuery = "insertOnDemandDataset"
const insertOnDemandDatasetMetadataFileQuery = "insertOnDemandDatasetMetadataFile"
const insertOnDemandDatasetCreatedFromOriginQuery = "insertOnDemandDatasetCreatedFromOrigin"

func init() {
	queries[insertOnDemandDatasetQuery] = `INSERT INTO on_demand_dataset (accession, requested_by_user, image_accessions_hash)
VALUES($1, $2, $3);`

	queries[insertOnDemandDatasetMetadataFileQuery] = `INSERT INTO on_demand_dataset_metadata_file (on_demand_dataset_accession, type, accession, xml_content)
VALUES($1, $2, $3, $4);`

	queries[insertOnDemandDatasetCreatedFromOriginQuery] = `INSERT INTO on_demand_dataset_created_from_origin (on_demand_dataset_accession, origin_accession)
VALUES($1, $2);`
}
func (db *pgDb) insertOnDemandDataset(ctx context.Context, tx *sql.Tx, onDemandDataset *models.OnDemandDataset, imageAccessionHash string) error {
	insertDatasetStmt, err := db.getPreparedStmt(tx, insertOnDemandDatasetQuery)
	if err != nil {
		return err
	}
	insertMetadataFileStmt, err := db.getPreparedStmt(tx, insertOnDemandDatasetMetadataFileQuery)
	if err != nil {
		return err
	}
	insertCreatedFromStmt, err := db.getPreparedStmt(tx, insertOnDemandDatasetCreatedFromOriginQuery)
	if err != nil {
		return err
	}

	if _, err := insertDatasetStmt.ExecContext(ctx,
		onDemandDataset.Accession,
		onDemandDataset.RequestedByUser,
		imageAccessionHash,
	); err != nil {
		return fmt.Errorf("failed to insert dod dataset: %w", err)
	}

	for _, createdFrom := range onDemandDataset.OriginDatasetAccessions {
		if _, err := insertCreatedFromStmt.ExecContext(ctx,
			onDemandDataset.Accession,
			createdFrom,
		); err != nil {
			return fmt.Errorf("failed to insert dod dataset created from origin: %w", err)
		}
	}

	datasetXml, err := xml.Marshal(onDemandDataset.DatasetMetadata.MetadataSet)
	if err != nil {
		return fmt.Errorf("failed to marshal dataset: %w", err)
	}

	if _, err := insertMetadataFileStmt.ExecContext(ctx,
		onDemandDataset.Accession,
		onDemandDataset.DatasetMetadata.MetadataFileType(),
		onDemandDataset.DatasetMetadata.Accession,
		datasetXml,
	); err != nil {
		return fmt.Errorf("failed to insert dataset metadata: %w", err)
	}

	imageXml, err := xml.Marshal(onDemandDataset.ImageMetadata.MetadataSet)
	if err != nil {
		return fmt.Errorf("failed to marshal image: %w", err)
	}
	if _, err := insertMetadataFileStmt.ExecContext(ctx,
		onDemandDataset.Accession,
		onDemandDataset.ImageMetadata.MetadataFileType(),
		onDemandDataset.ImageMetadata.Accession,
		imageXml,
	); err != nil {
		return fmt.Errorf("failed to insert image metadata: %w", err)
	}

	observationXml, err := xml.Marshal(onDemandDataset.ObservationMetadata.MetadataSet)
	if err != nil {
		return fmt.Errorf("failed to marshal observation: %w", err)
	}
	if _, err := insertMetadataFileStmt.ExecContext(ctx,
		onDemandDataset.Accession,
		onDemandDataset.ObservationMetadata.MetadataFileType(),
		onDemandDataset.ObservationMetadata.Accession,
		observationXml,
	); err != nil {
		return fmt.Errorf("failed to insert observation metadata: %w", err)
	}

	if onDemandDataset.ObserverMetadata != nil {
		observerXml, err := xml.Marshal(onDemandDataset.ObserverMetadata.MetadataSet)
		if err != nil {
			return fmt.Errorf("failed to marshal observer: %w", err)
		}
		if _, err := insertMetadataFileStmt.ExecContext(ctx,
			onDemandDataset.Accession,
			onDemandDataset.ObserverMetadata.MetadataFileType(),
			onDemandDataset.ObserverMetadata.Accession,
			observerXml,
		); err != nil {
			return fmt.Errorf("failed to insert observer metadata: %w", err)
		}
	}

	policyXml, err := xml.Marshal(onDemandDataset.PolicyMetadata.MetadataSet)
	if err != nil {
		return fmt.Errorf("failed to marshal policy: %w", err)
	}
	if _, err := insertMetadataFileStmt.ExecContext(ctx,
		onDemandDataset.Accession,
		onDemandDataset.PolicyMetadata.MetadataFileType(),
		onDemandDataset.PolicyMetadata.Accession,
		policyXml,
	); err != nil {
		return fmt.Errorf("failed to insert policy metadata: %w", err)
	}

	remsXml, err := xml.Marshal(onDemandDataset.RemsMetadata.MetadataSet)
	if err != nil {
		return fmt.Errorf("failed to marshal rems: %w", err)
	}
	if _, err := insertMetadataFileStmt.ExecContext(ctx,
		onDemandDataset.Accession,
		onDemandDataset.RemsMetadata.MetadataFileType(),
		onDemandDataset.RemsMetadata.Accession,
		remsXml,
	); err != nil {
		return fmt.Errorf("failed to insert rems metadata: %w", err)
	}

	sampleXml, err := xml.Marshal(onDemandDataset.SampleMetadata.MetadataSet)
	if err != nil {
		return fmt.Errorf("failed to marshal sample: %w", err)
	}
	if _, err := insertMetadataFileStmt.ExecContext(ctx,
		onDemandDataset.Accession,
		onDemandDataset.SampleMetadata.MetadataFileType(),
		onDemandDataset.SampleMetadata.Accession,
		sampleXml,
	); err != nil {
		return fmt.Errorf("failed to insert sample metadata: %w", err)
	}

	stainingXml, err := xml.Marshal(onDemandDataset.StainingMetadata.MetadataSet)
	if err != nil {
		return fmt.Errorf("failed to marshal staining: %w", err)
	}
	if _, err := insertMetadataFileStmt.ExecContext(ctx,
		onDemandDataset.Accession,
		onDemandDataset.StainingMetadata.MetadataFileType(),
		onDemandDataset.StainingMetadata.Accession,
		stainingXml,
	); err != nil {
		return fmt.Errorf("failed to insert staining metadata: %w", err)
	}

	return nil
}
