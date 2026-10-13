package models

import (
	"github.com/NBISweden/bp-dod-sda-gateway/models/metadata_models"
)

type DatasetMetadataFile[T MetadataSet] struct {
	DatasetAccession string
	MetadataFile     MetadataFile[T]
}
type MetadataFile[T MetadataSet] struct {
	Accession string
	// MetadataSet is one of the following
	// *metadata_models.DatasetSet
	// *metadata_models.ImageSet
	// *metadata_models.ObservationSet
	// *metadata_models.ObserverSet
	// *metadata_models.PolicySet
	// *metadata_models.RemsSet
	// *metadata_models.SampleSet
	// *metadata_models.StainingSet
	MetadataSet T
}

type MetadataSet interface {
	MetadataFileType() metadata_models.MetadataFileType
}

func (mf *MetadataFile[T]) MetadataFileType() metadata_models.MetadataFileType {
	return mf.MetadataSet.MetadataFileType()
}

type OnDemandDataset struct {
	Accession               string
	OriginDatasetAccessions []string
	RequestedByUser         string

	DatasetMetadata     MetadataFile[*metadata_models.DatasetSet]
	ImageMetadata       MetadataFile[*metadata_models.ImageSet]
	ObservationMetadata MetadataFile[*metadata_models.ObservationSet]
	ObserverMetadata    *MetadataFile[*metadata_models.ObserverSet] // Optional
	PolicyMetadata      MetadataFile[*metadata_models.PolicySet]
	RemsMetadata        MetadataFile[*metadata_models.RemsSet]
	SampleMetadata      MetadataFile[*metadata_models.SampleSet]
	StainingMetadata    MetadataFile[*metadata_models.StainingSet]
}
type OriginDataset struct {
	Accession          string
	RemsWorkflowID     int
	RemsOrganisationID string
	Dataset            *metadata_models.DatasetSet
	Image              *metadata_models.ImageSet
	// Annotation metadata is optional
	Annotation  *metadata_models.AnnotationSet
	Observation *metadata_models.ObservationSet
	// Observer metadata is optional
	Observer *metadata_models.ObserverSet
	Policy   *metadata_models.PolicySet
	Sample   *metadata_models.SampleSet
	Staining *metadata_models.StainingSet
}
