// Package metadata_models: file contains go representation of xsd structure: https://github.com/imi-bigpicture/bigpicture-metaflex/blob/2.0.0/src/BP.staining.xsd
package metadata_models

import (
	"encoding/xml"
	"errors"
)

type StainingSet struct {
	XMLName  xml.Name   `xml:"STAINING_SET"`
	Staining []Staining `xml:"STAINING"`
}
type Staining struct {
	ObjectType

	// Either ProcedureInformation or Stain needs to be present, but not both, and not none
	ProcedureInformation *Attributes `xml:"PROCEDURE_INFORMATION,omitempty"`
	Stain                *Attributes `xml:"STAIN,omitempty"`

	Attributes *NullableAttributes `xml:"ATTRIBUTES"`
}

func (x Staining) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	hasProcedureInformation := x.ProcedureInformation != nil
	hasStain := x.Stain != nil

	if hasProcedureInformation == hasStain { // both or neither
		return errors.New("exactly one of ProcedureInformation or Stain must be set")
	}

	type stainingAlias Staining // Avoid MarshalXML recursion by creating a new Staining type without MarshalXML

	return e.EncodeElement(stainingAlias(x), start)
}
