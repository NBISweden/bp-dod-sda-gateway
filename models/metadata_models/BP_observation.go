// Package metadata_models: file contains go representation of xsd structure: https://github.com/imi-bigpicture/bigpicture-metaflex/blob/2.0.0/src/BP.observation.xsd
package metadata_models

import "encoding/xml"

type ObservationSet struct {
	XMLName      xml.Name      `xml:"OBSERVATION_SET"`
	Observations []Observation `xml:"OBSERVATION"`
}

type Observation struct {
	ObjectType
	AnnotationRef      *Reference          `xml:"ANNOTATION_REF"`
	CaseRef            *Reference          `xml:"CASE_REF"`
	BiologicalBeingRef *Reference          `xml:"BIOLOGICAL_BEING_REF"`
	SpecimenRef        *Reference          `xml:"SPECIMEN_REF"`
	BlockRef           *Reference          `xml:"BLOCK_REF"`
	SlideRef           *Reference          `xml:"SLIDE_REF"`
	ImageRef           *Reference          `xml:"IMAGE_REF"`
	ObserverRef        []Reference         `xml:"OBSERVER_REF"`
	Statement          Statement           `xml:"STATEMENT"`
	Attributes         *NullableAttributes `xml:"ATTRIBUTES"`
}

type Statement struct {
	StatementType    string                  `xml:"STATEMENT_TYPE"`
	StatementStatus  string                  `xml:"STATEMENT_STATUS"`
	CodeAttributes   *NullableCodeAttributes `xml:"CODE_ATTRIBUTES"`
	CustomAttributes *NullableAttributes     `xml:"CUSTOM_ATTRIBUTES"`
	Freetext         *NullableString         `xml:"FREETEXT"`
	Attributes       *NullableAttributes     `xml:"ATTRIBUTES"`
}
