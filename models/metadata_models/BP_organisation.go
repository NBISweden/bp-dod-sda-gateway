// Package metadata_models: file contains go representation of xsd structure: https://github.com/imi-bigpicture/bigpicture-metaflex/blob/2.0.0/src/BP.organisation.xsd
package metadata_models

import "encoding/xml"

type OrganisationSet struct {
	XMLName       xml.Name       `xml:"ORGANISATION_SET"`
	Organisations []Organisation `xml:"ORGANISATION"`
}
type Organisation struct {
	ObjectType
	Name                  string              `xml:"NAME"`
	PicCode               int                 `xml:"PIC_CODE"`
	DatamanagerPerunGroup string              `xml:"DATAMANAGER_PERUN_GROUP"`
	DatasetRef            Reference           `xml:"DATASET_REF"`
	Attributes            *NullableAttributes `xml:"ATTRIBUTES"`
}
