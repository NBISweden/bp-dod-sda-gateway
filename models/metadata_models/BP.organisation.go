package metadata_models

import "encoding/xml"

type OrganisationSet struct {
	XMLName  xml.Name `xml:"ORGANISATION_SET"`
	Policies []Policy `xml:"ORGANISATION"`
}
type Organisation struct {
	ObjectType
	Name                  string              `xml:"NAME"`
	PicCode               int                 `xml:"PIC_CODE"`
	DatamanagerPerunGroup string              `xml:"DATAMANAGER_PERUN_GROUP"`
	DatasetRef            Reference           `xml:"DATASET_REF"`
	Attributes            *NullableAttributes `xml:"ATTRIBUTES"`
}
