package metadata_models

import "encoding/xml"

type LandingPageSet struct {
	XMLName      xml.Name      `xml:"LANDING_PAGE_SET"`
	LandingPages []LandingPage `xml:"LANDING_PAGE"`
}

type LandingPage struct {
	ObjectType
	DatasetRef     Reference           `xml:"DATASET_REF"`
	RemsAccessLink *string             `xml:"REMS_ACCESS_LINK"`
	ObserverRef    []SampleImageFiles  `xml:"SAMPLE_IMAGE_FILES"`
	Attributes     *NullableAttributes `xml:"ATTRIBUTES"`
}

type SampleImageFiles struct {
	FileBaseType
	Filetype string `xml:"filetype,attr"`
}
