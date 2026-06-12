package metadata_models

import "encoding/xml"

type ObserverSet struct {
	XMLName   xml.Name   `xml:"OBSERVER_SET"`
	Observers []Observer `xml:"OBSERVER"`
}

type Observer struct {
	ObjectType
	ObserverType string              `xml:"OBSERVER_TYPE"`
	Attributes   *NullableAttributes `xml:"ATTRIBUTES"`
}
