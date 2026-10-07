// Package metadata_models: file contains go representation of xsd structure: https://github.com/imi-bigpicture/bigpicture-metaflex/blob/2.0.0/src/BP.rems.xsd
package metadata_models

import "encoding/xml"

type RemsSet struct {
	XMLName xml.Name `xml:"REMS_SET"`
	Rems    []Rems   `xml:"REMS"`
}
type Rems struct {
	ObjectType
	WorkflowId     string              `xml:"WORKFLOW_ID"`
	OrganisationId string              `xml:"ORGANISATION_ID"`
	DatasetRef     Reference           `xml:"DATASET_REF"`
	Attributes     *NullableAttributes `xml:"ATTRIBUTES"`
}
