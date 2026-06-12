package metadata_models

import "encoding/xml"

type AnnotationSet struct {
	XMLName    xml.Name     `xml:"ANNOTATION_SET"`
	Annotation []Annotation `xml:"ANNOTATION"`
}

type Annotation struct {
	ObjectType
	ImageRef   Reference          `xml:"IMAGE_REF"`
	Files      AnnotationFiles    `xml:"FILES"`
	Attributes NullableAttributes `xml:"ATTRIBUTES"`
}
type AnnotationFiles struct {
	Files []ImageFile `xml:"FILE"`
}

type AnnotationFile struct {
	FileBaseType
	Filename string `xml:"filetype,attr"`
}
