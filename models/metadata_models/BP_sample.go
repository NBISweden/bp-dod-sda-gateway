// Package metadata_models: file contains go representation of xsd structure: https://github.com/imi-bigpicture/bigpicture-metaflex/blob/2.0.0/src/BP.sample.xsd
package metadata_models

import "encoding/xml"

type SampleSet struct {
	XMLName          xml.Name          `xml:"SAMPLE_SET"`
	BiologicalBeings []BiologicalBeing `xml:"BIOLOGICAL_BEING"`
	Cases            []Case            `xml:"CASE"`
	Specimens        []Specimen        `xml:"SPECIMEN"`
	Blocks           []Block           `xml:"BLOCK"`
	Slides           []Slide           `xml:"SLIDE"`
}

type BiologicalBeing struct {
	ObjectType
	Attributes NullableAttributes `xml:"ATTRIBUTES"`
}

type Case struct {
	ObjectType
	BiologicalBeingRef Reference           `xml:"BIOLOGICAL_BEING_REF"`
	Attributes         *NullableAttributes `xml:"ATTRIBUTES"`
}

type Specimen struct {
	ObjectType
	ExtractedFromRef Reference           `xml:"EXTRACTED_FROM_REF"`
	PartOfCaseRef    *Reference          `xml:"PART_OF_CASE_REF"`
	Attributes       *NullableAttributes `xml:"ATTRIBUTES"`
}

type Block struct {
	ObjectType
	SampledFromRef []Reference         `xml:"SAMPLED_FROM_REF"`
	Attributes     *NullableAttributes `xml:"ATTRIBUTES"`
}

type Slide struct {
	ObjectType
	CreatedFromRef         Reference           `xml:"CREATED_FROM_REF"`
	StainingInformationRef Reference           `xml:"STAINING_INFORMATION_REF"`
	Attributes             *NullableAttributes `xml:"ATTRIBUTES"`
}
