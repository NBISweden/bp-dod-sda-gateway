package metadata_models

import "encoding/xml"

type RemsSet struct {
	XMLName xml.Name `xml:"REMS_SET"`
	Rems    []Rems   `json:"REMS"`
}
type Rems struct {
	ObjectType
	WorkflowId     string              `xml:"WORKFLOW_ID"`
	OrganisationId string              `xml:"ORGANISATION_ID"`
	DatasetRef     Reference           `xml:"DATASET_REF"`
	Attributes     *NullableAttributes `xml:"ATTRIBUTES"`
}
