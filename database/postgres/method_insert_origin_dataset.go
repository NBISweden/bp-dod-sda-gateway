package postgres

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"

	"github.com/NBISweden/bp-dod-sda-gateway/models"
)

const insertOriginDatasetQuery = "insertOriginDataset"

func init() {
	queries[insertOriginDatasetQuery] = `INSERT INTO origin_dataset (accession, rems_workflow_id, rems_organisation_id, dataset_xml, image_xml, annotation_xml, observation_xml, observer_xml, policy_xml, sample_xml, staining_xml)
VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);`
}
func (db *pgDb) insertOriginDataset(ctx context.Context, tx *sql.Tx, originDataset *models.OriginDataset) error {
	stmt, err := db.getPreparedStmt(tx, insertOriginDatasetQuery)
	if err != nil {
		return err
	}

	datasetXml, err := xml.Marshal(originDataset.Dataset)
	if err != nil {
		return fmt.Errorf("failed to marshal dataset: %w", err)
	}
	imageXml, err := xml.Marshal(originDataset.Image)
	if err != nil {
		return fmt.Errorf("failed to marshal image: %w", err)
	}
	var annotationXml sql.Null[[]byte]
	if originDataset.Annotation != nil {
		annotationXmlString, err := xml.Marshal(originDataset.Annotation)
		if err != nil {
			return fmt.Errorf("failed to marshal annotation: %w", err)
		}
		annotationXml = sql.Null[[]byte]{
			V:     annotationXmlString,
			Valid: true,
		}
	}

	observationXml, err := xml.Marshal(originDataset.Observation)
	if err != nil {
		return fmt.Errorf("failed to marshal observation: %w", err)
	}
	var observerXml sql.Null[[]byte]
	if originDataset.Observer != nil {
		observerXmlString, err := xml.Marshal(originDataset.Observer)
		if err != nil {
			return fmt.Errorf("failed to marshal observer: %w", err)
		}
		observerXml = sql.Null[[]byte]{
			V:     observerXmlString,
			Valid: true,
		}
	}

	policyXml, err := xml.Marshal(originDataset.Policy)
	if err != nil {
		return fmt.Errorf("failed to marshal policy: %w", err)
	}
	sampleXml, err := xml.Marshal(originDataset.Sample)
	if err != nil {
		return fmt.Errorf("failed to marshal sample: %w", err)
	}
	stainingXml, err := xml.Marshal(originDataset.Staining)
	if err != nil {
		return fmt.Errorf("failed to marshal staining: %w", err)
	}

	result, err := stmt.ExecContext(ctx,
		originDataset.Accession,
		originDataset.RemsWorkflowID,
		originDataset.RemsOrganisationID,
		datasetXml,
		imageXml,
		annotationXml,
		observationXml,
		observerXml,
		policyXml,
		sampleXml,
		stainingXml,
	)
	if err != nil {
		return err
	}
	if rowsAffected, _ := result.RowsAffected(); rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
