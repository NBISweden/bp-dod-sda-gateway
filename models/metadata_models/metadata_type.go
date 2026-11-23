package metadata_models

import (
	"database/sql/driver"
	"fmt"
)

type MetadataFileType uint8

const (
	MetadataFileTypeInvalid MetadataFileType = iota
	MetadataFileTypeDataset
	MetadataFileTypeImage
	MetadataFileTypeAnnotation
	MetadataFileTypeObservation
	MetadataFileTypeObserver
	MetadataFileTypePolicy
	MetadataFileTypeRems
	MetadataFileTypeSample
	MetadataFileTypeStaining
)

func (t MetadataFileType) String() string {
	switch t {
	case MetadataFileTypeDataset:
		return "dataset"
	case MetadataFileTypeImage:
		return "image"
	case MetadataFileTypeAnnotation:
		return "annotation"
	case MetadataFileTypeObservation:
		return "observation"
	case MetadataFileTypeObserver:
		return "observer"
	case MetadataFileTypePolicy:
		return "policy"
	case MetadataFileTypeRems:
		return "rems"
	case MetadataFileTypeSample:
		return "sample"
	case MetadataFileTypeStaining:
		return "staining"
	default:
		return ""
	}
}

func ParseMetadataFileType(s string) MetadataFileType {
	switch s {
	case "dataset":
		return MetadataFileTypeDataset
	case "image":
		return MetadataFileTypeImage
	case "annotation":
		return MetadataFileTypeAnnotation
	case "observation":
		return MetadataFileTypeObservation
	case "observer":
		return MetadataFileTypeObserver
	case "policy":
		return MetadataFileTypePolicy
	case "rems":
		return MetadataFileTypeRems
	case "sample":
		return MetadataFileTypeSample
	case "staining":
		return MetadataFileTypeStaining
	default:
		return MetadataFileTypeInvalid
	}
}

func (t MetadataFileType) Value() (driver.Value, error) {
	return t.String(), nil
}

func (t *MetadataFileType) Scan(src any) error {
	var s string

	switch v := src.(type) {
	case string:
		s = v
	case []byte:
		s = string(v)
	case nil:
		*t = MetadataFileTypeInvalid

		return nil
	default:
		return fmt.Errorf("cannot scan %T into MetadataFileType", src)
	}

	*t = ParseMetadataFileType(s)

	return nil
}

func (*DatasetSet) MetadataFileType() MetadataFileType {
	return MetadataFileTypeDataset
}

func (*ImageSet) MetadataFileType() MetadataFileType {
	return MetadataFileTypeImage
}

func (*ObservationSet) MetadataFileType() MetadataFileType {
	return MetadataFileTypeObservation
}
func (*ObserverSet) MetadataFileType() MetadataFileType {
	return MetadataFileTypeObserver
}
func (*PolicySet) MetadataFileType() MetadataFileType {
	return MetadataFileTypePolicy
}
func (*RemsSet) MetadataFileType() MetadataFileType {
	return MetadataFileTypeRems
}
func (*SampleSet) MetadataFileType() MetadataFileType {
	return MetadataFileTypeSample
}
func (*StainingSet) MetadataFileType() MetadataFileType {
	return MetadataFileTypeStaining
}
