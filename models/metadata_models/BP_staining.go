// Package metadata_models: file contains go representation of xsd structure: https://github.com/imi-bigpicture/bigpicture-metaflex/blob/2.0.0/src/BP.staining.xsd
package metadata_models

import "encoding/xml"

type StainingSet struct {
	XMLName  xml.Name   `xml:"STAINING_SET"`
	Staining []Staining `xml:"STAINING"`
}
type Staining struct {
	ObjectType
	Attributes           *NullableAttributes `xml:"ATTRIBUTES"`
	ProcedureInformation Attributes          `xml:"PROCEDURE_INFORMATION"`
	Stain                Attributes          `xml:"STAIN"`
}
