package metadata_submitter_database

import (
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"

	"github.com/NBISweden/bp-dod-sda-gateway/models/metadata_models"
	"github.com/NBISweden/bp-dod-sda-gateway/origin_dataset_file_loader"
)

const getMetadataXmlQuery = "getMetadataXml"

func init() {
	queries[getMetadataXmlQuery] = `
SELECT xml_document, object_type
FROM objects
WHERE submission_id = $1
AND (
  ($2 = 'dataset' AND object_type = 'dataset')
  OR
  ($2 = 'image' AND object_type = 'image')
  OR
  ($2 = 'annotation' AND object_type = 'annotation')
  OR
  ($2 = 'observation' AND object_type = 'observation')
  OR
  ($2 = 'observer' AND object_type = 'observer')
  OR
  ($2 = 'policy' AND object_type = 'policy')
  OR
  ($2 = 'sample' AND object_type IN (
      'slide',
      'block',
      'case',
      'specimen',
      'biological_being'
  ))
  OR 
  ($2 = 'staining' AND object_type = 'staining')
);
`
}
func (db *metadataSubmitterPg) unmarshalFileToXml(ctx context.Context, file *origin_dataset_file_loader.FileInfo, dst any) error {
	stmt, err := db.getPreparedStmt(getMetadataXmlQuery)
	if err != nil {
		return err
	}

	rows, err := stmt.QueryContext(ctx,
		file.DatasetAccession,
		file.MetadataFileType,
	)
	if err != nil {
		return err
	}
	defer func() {
		_ = rows.Close()
	}()

	var rowsFound bool
	for rows.Next() {
		rowsFound = true
		var metadataSetEntryXmlContent []byte
		var entryType string
		if err := rows.Scan(&metadataSetEntryXmlContent, &entryType); err != nil {
			return err
		}

		switch entryType {
		case "dataset":
			var datasetEntry metadata_models.Dataset
			if err := xml.Unmarshal(metadataSetEntryXmlContent, &datasetEntry); err != nil {
				return fmt.Errorf("failed to unmarshal dataset: %w", err)
			}

			dst.(*metadata_models.DatasetSet).Dataset = append(dst.(*metadata_models.DatasetSet).Dataset, datasetEntry)
		case "image":
			var imageEntry metadata_models.Image
			if err := xml.Unmarshal(metadataSetEntryXmlContent, &imageEntry); err != nil {
				return fmt.Errorf("failed to unmarshal image: %w", err)
			}

			dst.(*metadata_models.ImageSet).Images = append(dst.(*metadata_models.ImageSet).Images, imageEntry)
		case "annotation":
			var annotationEntry metadata_models.Annotation
			if err := xml.Unmarshal(metadataSetEntryXmlContent, &annotationEntry); err != nil {
				return fmt.Errorf("failed to unmarshal annotation: %w", err)
			}

			dst.(*metadata_models.AnnotationSet).Annotation = append(dst.(*metadata_models.AnnotationSet).Annotation, annotationEntry)
		case "observation":
			var observationEntry metadata_models.Observation
			if err := xml.Unmarshal(metadataSetEntryXmlContent, &observationEntry); err != nil {
				return fmt.Errorf("failed to unmarshal observation: %w", err)
			}

			dst.(*metadata_models.ObservationSet).Observations = append(dst.(*metadata_models.ObservationSet).Observations, observationEntry)
		case "observer":
			var observerEntry metadata_models.Observer
			if err := xml.Unmarshal(metadataSetEntryXmlContent, &observerEntry); err != nil {
				return fmt.Errorf("failed to unmarshal observer: %w", err)
			}

			dst.(*metadata_models.ObserverSet).Observers = append(dst.(*metadata_models.ObserverSet).Observers, observerEntry)
		case "policy":
			var policyEntry metadata_models.Policy
			if err := xml.Unmarshal(metadataSetEntryXmlContent, &policyEntry); err != nil {
				return fmt.Errorf("failed to unmarshal policy: %w", err)
			}

			dst.(*metadata_models.PolicySet).Policies = append(dst.(*metadata_models.PolicySet).Policies, policyEntry)
		case "slide":
			var slideEntry metadata_models.Slide
			if err := xml.Unmarshal(metadataSetEntryXmlContent, &slideEntry); err != nil {
				return fmt.Errorf("failed to unmarshal slide: %w", err)
			}

			dst.(*metadata_models.SampleSet).Slides = append(dst.(*metadata_models.SampleSet).Slides, slideEntry)
		case "block":
			var blockEntry metadata_models.Block
			if err := xml.Unmarshal(metadataSetEntryXmlContent, &blockEntry); err != nil {
				return fmt.Errorf("failed to unmarshal block: %w", err)
			}

			dst.(*metadata_models.SampleSet).Blocks = append(dst.(*metadata_models.SampleSet).Blocks, blockEntry)
		case "case":
			var caseEntry metadata_models.Case
			if err := xml.Unmarshal(metadataSetEntryXmlContent, &caseEntry); err != nil {
				return fmt.Errorf("failed to unmarshal case: %w", err)
			}

			dst.(*metadata_models.SampleSet).Cases = append(dst.(*metadata_models.SampleSet).Cases, caseEntry)
		case "specimen":
			var specimenEntry metadata_models.Specimen
			if err := xml.Unmarshal(metadataSetEntryXmlContent, &specimenEntry); err != nil {
				return fmt.Errorf("failed to unmarshal specimen: %w", err)
			}

			dst.(*metadata_models.SampleSet).Specimens = append(dst.(*metadata_models.SampleSet).Specimens, specimenEntry)
		case "biological_being":
			var biologicalBeingEntry metadata_models.BiologicalBeing
			if err := xml.Unmarshal(metadataSetEntryXmlContent, &biologicalBeingEntry); err != nil {
				return fmt.Errorf("failed to unmarshal biological being: %w", err)
			}

			dst.(*metadata_models.SampleSet).BiologicalBeings = append(dst.(*metadata_models.SampleSet).BiologicalBeings, biologicalBeingEntry)
		case "staining":
			var stainingEntry metadata_models.Staining
			if err := xml.Unmarshal(metadataSetEntryXmlContent, &stainingEntry); err != nil {
				return fmt.Errorf("failed to unmarshal staining: %w", err)
			}

			dst.(*metadata_models.StainingSet).Staining = append(dst.(*metadata_models.StainingSet).Staining, stainingEntry)
		default:
			return errors.New("unknown metadata entry")
		}
	}

	if err := rows.Err(); err != nil {
		return err
	}

	if !rowsFound {
		return sql.ErrNoRows
	}

	return nil
}
