package metadata_models

import "encoding/xml"

type ImageSet struct {
	XMLName xml.Name `xml:"IMAGE_SET"`
	Images  []Image  `xml:"IMAGE"`
}

type Image struct {
	ObjectType
	// ImageOf is specified as maxOccurs="unbounded" in xsd files, but in Metadata Standards v2.0.0, its described to only support 1 value.
	ImageOf    Reference          `xml:"IMAGE_OF"`
	ImageType  ImageTypeChoice    `xml:"IMAGE_TYPE"`
	Files      ImageFiles         `xml:"FILES"`
	Attributes NullableAttributes `xml:"ATTRIBUTES"`
}

type ImageFile struct {
	FileBaseType
	FileType string `xml:"filetype,attr"`
}

type ImageFiles struct {
	Files []ImageFile `xml:"FILE"`
}

type ImageTypeChoice struct {
	WSIImage   *string `xml:"WSI_IMAGE,omitempty"`
	GrossImage *string `xml:"GROSS_IMAGE,omitempty"`
}
