// Package metadata_models: file contains go representation of xsd structure: https://github.com/imi-bigpicture/bigpicture-metaflex/blob/2.0.0/src/BP.observer.xsd
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
