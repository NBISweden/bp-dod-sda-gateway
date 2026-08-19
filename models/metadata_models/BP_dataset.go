// Package metadata_models: file contains go representation of xsd structure: https://github.com/imi-bigpicture/bigpicture-metaflex/blob/2.0.0/src/BP.dataset.xsd
package metadata_models

import "encoding/xml"

type DatasetSet struct {
	XMLName xml.Name  `xml:"DATASET_SET"`
	Dataset []Dataset `xml:"DATASET"`
}

type Dataset struct {
	ObjectType
	Title                    string              `xml:"TITLE"`
	ShortName                string              `xml:"SHORT_NAME"`
	Description              *string             `xml:"DESCRIPTION"`
	Version                  string              `xml:"VERSION"`
	MetadataStandard         string              `xml:"METADATA_STANDARD"`
	DatasetOwnerContactEmail *string             `xml:"DATASET_OWNER_CONTACT_EMAIL"`
	DatasetType              []string            `xml:"DATASET_TYPE"`
	ImageRef                 []Reference         `xml:"IMAGE_REF"`
	AnnotationRef            []Reference         `xml:"ANNOTATION_REF"`
	ObservationRef           []Reference         `xml:"OBSERVATION_REF"`
	ComplementsDatasetRef    []Reference         `xml:"COMPLEMENTS_DATASET_REF"`
	Attributes               *NullableAttributes `xml:"ATTRIBUTES"`
}
