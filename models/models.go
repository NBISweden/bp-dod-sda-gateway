package models

import (
	"github.com/NBISweden/bp-dod-sda-gateway/models/metadata_models"
)

//
//type MetadataFileStatus uint8
//
//const (
//	MetadataFileStatusInvalid MetadataFileStatus = iota
//	MetadataFileStatusRequested
//	MetadataFileStatusUploaded
//	MetadataFileStatusVerified
//	MetadataFileStatusAccession
//	MetadataFileStatusReady
//)
//
//func (s MetadataFileStatus) String() string {
//	switch s {
//	case MetadataFileStatusRequested:
//		return "requested"
//	case MetadataFileStatusUploaded:
//		return "uploaded"
//	case MetadataFileStatusVerified:
//		return "verified"
//	case MetadataFileStatusAccession:
//		return "accession"
//	case MetadataFileStatusReady:
//		return "ready"
//	default:
//		return ""
//	}
//}
//func ParseMetadataFileStatus(s string) MetadataFileStatus {
//	switch s {
//	case "requested":
//		return MetadataFileStatusRequested
//	case "uploaded":
//		return MetadataFileStatusUploaded
//	case "verified":
//		return MetadataFileStatusVerified
//	case "accession":
//		return MetadataFileStatusAccession
//	case "ready":
//		return MetadataFileStatusReady
//	default:
//		return MetadataFileStatusInvalid
//	}
//}
//func (s MetadataFileStatus) Value() (driver.Value, error) {
//	return s.String(), nil
//}
//func (s *MetadataFileStatus) Scan(src any) error {
//	var str string
//
//	switch v := src.(type) {
//	case string:
//		str = v
//	case []byte:
//		str = string(v)
//	case nil:
//		*s = MetadataFileStatusInvalid
//		return nil
//	default:
//		return fmt.Errorf("cannot scan %T into MetadataFileStatus", src)
//	}
//
//	*s = ParseMetadataFileStatus(str)
//
//	return nil
//}

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

type DatasetOnDemandDataset struct {
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
