package postgres

import (
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"

	"github.com/NBISweden/bp-dod-sda-gateway/models"
	"github.com/NBISweden/bp-dod-sda-gateway/pkg/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const getOriginDatasetQuery = "getOriginDataset"

func init() {
	queries[getOriginDatasetQuery] = `SELECT accession, rems_workflow_id, rems_organisation_id, dataset_xml, image_xml, annotation_xml, observation_xml, observer_xml, policy_xml, sample_xml, staining_xml
FROM origin_dataset
WHERE accession = $1

`
}
func (db *pgDb) getOriginDataset(ctx context.Context, tx *sql.Tx, accession string) (*models.OriginDataset, error) {
	ctx, span := observability.Tracer().Start(ctx, "getOriginDataset", trace.WithAttributes(attribute.String("accession", accession)))
	defer span.End()

	stmt, err := db.getPreparedStmt(tx, getOriginDatasetQuery)
	if err != nil {
		return nil, err
	}

	originDataset := new(models.OriginDataset)

	var observerXml, annotationXml sql.Null[[]byte]

	var datasetXml, imageXml, observationXml, policyXml, sampleXml, stainingXml []byte

	if err := stmt.QueryRowContext(ctx, accession).Scan(
		&originDataset.Accession,
		&originDataset.RemsWorkflowID,
		&originDataset.RemsOrganisationID,
		&datasetXml,
		&imageXml,
		&annotationXml,
		&observationXml,
		&observerXml,
		&policyXml,
		&sampleXml,
		&stainingXml,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	if err := xml.Unmarshal(datasetXml, &originDataset.Dataset); err != nil {
		return nil, fmt.Errorf("failed to marshal dataset: %w", err)
	}

	if err := xml.Unmarshal(imageXml, &originDataset.Image); err != nil {
		return nil, fmt.Errorf("failed to unmarshal image: %w", err)
	}
	if annotationXml.Valid {
		if err := xml.Unmarshal(annotationXml.V, &originDataset.Annotation); err != nil {
			return nil, fmt.Errorf("failed to marshal annotation: %w", err)
		}
	}

	if err := xml.Unmarshal(observationXml, &originDataset.Observation); err != nil {
		return nil, fmt.Errorf("failed to marshal observation: %w", err)
	}
	if observerXml.Valid {
		if err := xml.Unmarshal(observerXml.V, &originDataset.Observer); err != nil {
			return nil, fmt.Errorf("failed to marshal observer: %w", err)
		}
	}

	if err := xml.Unmarshal(policyXml, &originDataset.Policy); err != nil {
		return nil, fmt.Errorf("failed to marshal policy: %w", err)
	}
	if err := xml.Unmarshal(sampleXml, &originDataset.Sample); err != nil {
		return nil, fmt.Errorf("failed to marshal sample: %w", err)
	}
	if err := xml.Unmarshal(stainingXml, &originDataset.Staining); err != nil {
		return nil, fmt.Errorf("failed to marshal staining: %w", err)
	}

	return originDataset, nil
}
