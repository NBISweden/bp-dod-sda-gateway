// Package metadata_models file contains go representation of xsd structure: https://github.com/imi-bigpicture/bigpicture-metaflex/blob/2.0.0/src/BP.annotation.xsd
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
	Files []AnnotationFile `xml:"FILE"`
}

type AnnotationFile struct {
	FileBaseType
	FileType string `xml:"filetype,attr"`
}
