package metadata_models

import "encoding/xml"

type PolicySet struct {
	XMLName  xml.Name `xml:"POLICY_SET"`
	Policies []Policy `xml:"POLICY"`
}

type Policy struct {
	ObjectType
	DatasetRef Reference          `xml:"DATASET_REF"`
	Attributes NullableAttributes `xml:"ATTRIBUTES"`
}
