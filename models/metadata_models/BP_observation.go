// Package metadata_models: file contains go representation of xsd structure: https://github.com/imi-bigpicture/bigpicture-metaflex/blob/2.0.0/src/BP.observation.xsd
package metadata_models

import "encoding/xml"

type ObservationSet struct {
	XMLName      xml.Name      `xml:"OBSERVATION_SET"`
	Observations []Observation `xml:"OBSERVATION"`
}

type Observation struct {
	ObjectType
	// Observation is either linked to an AnnotationRef, CaseRef, BiologicalBeingRef, SpecimenRef, BlockRef, SlideRef, ImageRef
	AnnotationRef      *Reference          `xml:"ANNOTATION_REF,omitempty"`
	CaseRef            *Reference          `xml:"CASE_REF,omitempty"`
	BiologicalBeingRef *Reference          `xml:"BIOLOGICAL_BEING_REF,omitempty"`
	SpecimenRef        *Reference          `xml:"SPECIMEN_REF,omitempty"`
	BlockRef           *Reference          `xml:"BLOCK_REF,omitempty"`
	SlideRef           *Reference          `xml:"SLIDE_REF,omitempty"`
	ImageRef           *Reference          `xml:"IMAGE_REF,omitempty"`
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
