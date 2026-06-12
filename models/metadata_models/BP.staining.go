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
