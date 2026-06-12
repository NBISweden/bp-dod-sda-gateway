package metadata_models

import (
	"encoding/xml"
	"strconv"
)

type Reference struct {
	Alias     string `xml:"alias,attr"`
	Accession string `xml:"accession,attr,omitempty"`
}

type FileBaseType struct {
	Filename            string `xml:"filename,attr"`
	ChecksumMethod      string `xml:"checksum_method,attr"`
	Checksum            string `xml:"checksum,attr"`
	UnencryptedChecksum string `xml:"unencrypted_checksum,attr,omitempty"`
}

type ObjectType struct {
	Alias     string `xml:"alias,attr"`
	Accession string `xml:"accession,attr,omitempty"`
}

type Attributes struct {
	StringAttributes      []StringAttribute      `xml:"STRING_ATTRIBUTE"`
	NumericAttributes     []NumericAttribute     `xml:"NUMERIC_ATTRIBUTE"`
	MeasurementAttributes []MeasurementAttribute `xml:"MEASUREMENT_ATTRIBUTE"`
	CodeAttributes        []CodeAttribute        `xml:"CODE_ATTRIBUTE"`
	SetAttributes         []SetAttribute         `xml:"SET_ATTRIBUTE"`
}

type StringAttribute struct {
	Tag   string          `xml:"TAG"`
	Value *NullableString `xml:"VALUE"`
}

type NumericAttribute struct {
	Tag   string         `xml:"TAG"`
	Value *NullableFloat `xml:"VALUE"`
}

type MeasurementAttribute struct {
	Tag   string         `xml:"TAG"`
	Value *NullableFloat `xml:"VALUE"`
	Units string         `xml:"UNITS"`
}

type CodeAttribute struct {
	Tag   string                      `xml:"TAG"`
	Value *NullableCodeAttributeValue `xml:"VALUE"`
}

type SetAttribute struct {
	Tag   string              `xml:"TAG"`
	Value *NullableAttributes `xml:"VALUE"`
}

type CodeAttributeValue struct {
	Code          string          `xml:"CODE"`
	Scheme        string          `xml:"SCHEME"`
	Meaning       string          `xml:"MEANING"`
	SchemeVersion *NullableString `xml:"SCHEME_VERSION"`
}

type CodeAttributes struct {
	CodeAttributes []CodeAttribute `xml:"CODE_ATTRIBUTE"`
}

type NullableString struct {
	Value *string
	Nil   bool
}

func (n *NullableString) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if n.Nil {
		start.Attr = append(start.Attr,
			xml.Attr{
				Name: xml.Name{
					Local: "xmlns:xsi",
				},
				Value: "http://www.w3.org/2001/XMLSchema-instance",
			},
			xml.Attr{
				Name: xml.Name{
					Local: "xsi:nil",
				},
				Value: "true",
			},
		)

		if err := e.EncodeToken(start); err != nil {
			return err
		}
		return e.EncodeToken(start.End())
	}

	if n.Value == nil {
		return e.EncodeElement(nil, start)
	}

	return e.EncodeElement(*n.Value, start)
}

func (n *NullableString) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, a := range start.Attr {
		if a.Name.Space == "http://www.w3.org/2001/XMLSchema-instance" &&
			a.Name.Local == "nil" &&
			a.Value == "true" {

			n.Nil = true

			// consume element
			var discard any
			return d.DecodeElement(&discard, &start)
		}
	}

	var v string
	if err := d.DecodeElement(&v, &start); err != nil {
		return err
	}

	n.Value = &v
	return nil
}

type NullableFloat struct {
	Value *float64
	Nil   bool
}

func (n *NullableFloat) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if n.Nil {
		start.Attr = append(start.Attr,
			xml.Attr{
				Name: xml.Name{
					Local: "xmlns:xsi",
				},
				Value: "http://www.w3.org/2001/XMLSchema-instance",
			},
			xml.Attr{
				Name: xml.Name{
					Local: "xsi:nil",
				},
				Value: "true",
			},
		)

		if err := e.EncodeToken(start); err != nil {
			return err
		}
		return e.EncodeToken(start.End())
	}

	if n.Value == nil {
		return e.EncodeElement(nil, start)
	}

	var s string
	if *n.Value == 0 {
		s = "0.0"
	} else {
		s = strconv.FormatFloat(*n.Value, 'g', -1, 64)
	}

	return e.EncodeElement(s, start)
}

func (n *NullableFloat) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, a := range start.Attr {
		if a.Name.Space == "http://www.w3.org/2001/XMLSchema-instance" &&
			a.Name.Local == "nil" &&
			a.Value == "true" {

			n.Nil = true

			// consume element
			var discard any
			return d.DecodeElement(&discard, &start)
		}
	}

	var v float64
	if err := d.DecodeElement(&v, &start); err != nil {
		return err
	}

	n.Value = &v
	return nil
}

type NullableAttributes struct {
	Value *Attributes
	Nil   bool
}

func (n *NullableAttributes) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if n.Nil {
		start.Attr = append(start.Attr,
			xml.Attr{
				Name: xml.Name{
					Local: "xmlns:xsi",
				},
				Value: "http://www.w3.org/2001/XMLSchema-instance",
			},
			xml.Attr{
				Name: xml.Name{
					Local: "xsi:nil",
				},
				Value: "true",
			},
		)

		if err := e.EncodeToken(start); err != nil {
			return err
		}
		return e.EncodeToken(start.End())
	}

	if n.Value == nil {
		return e.EncodeElement(nil, start)
	}

	return e.EncodeElement(*n.Value, start)
}

func (n *NullableAttributes) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, a := range start.Attr {
		if a.Name.Space == "http://www.w3.org/2001/XMLSchema-instance" &&
			a.Name.Local == "nil" &&
			a.Value == "true" {

			n.Nil = true

			// consume element
			var discard any
			return d.DecodeElement(&discard, &start)
		}
	}

	var v Attributes
	if err := d.DecodeElement(&v, &start); err != nil {
		return err
	}

	n.Value = &v
	return nil
}

type NullableCodeAttributeValue struct {
	Value *CodeAttributeValue
	Nil   bool
}

func (n *NullableCodeAttributeValue) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if n.Nil {
		start.Attr = append(start.Attr,
			xml.Attr{
				Name: xml.Name{
					Local: "xmlns:xsi",
				},
				Value: "http://www.w3.org/2001/XMLSchema-instance",
			},
			xml.Attr{
				Name: xml.Name{
					Local: "xsi:nil",
				},
				Value: "true",
			},
		)

		if err := e.EncodeToken(start); err != nil {
			return err
		}
		return e.EncodeToken(start.End())
	}

	if n.Value == nil {
		return e.EncodeElement(nil, start)
	}

	return e.EncodeElement(*n.Value, start)
}

func (n *NullableCodeAttributeValue) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, a := range start.Attr {
		if a.Name.Space == "http://www.w3.org/2001/XMLSchema-instance" &&
			a.Name.Local == "nil" &&
			a.Value == "true" {

			n.Nil = true

			// consume element
			var discard any
			return d.DecodeElement(&discard, &start)
		}
	}

	var v CodeAttributeValue
	if err := d.DecodeElement(&v, &start); err != nil {
		return err
	}

	n.Value = &v
	return nil
}

type NullableCodeAttributes struct {
	Value *CodeAttributes
	Nil   bool
}

func (n *NullableCodeAttributes) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if n.Nil {
		start.Attr = append(start.Attr,
			xml.Attr{
				Name: xml.Name{
					Local: "xmlns:xsi",
				},
				Value: "http://www.w3.org/2001/XMLSchema-instance",
			},
			xml.Attr{
				Name: xml.Name{
					Local: "xsi:nil",
				},
				Value: "true",
			},
		)

		if err := e.EncodeToken(start); err != nil {
			return err
		}
		return e.EncodeToken(start.End())
	}

	if n.Value == nil {
		return e.EncodeElement(nil, start)
	}

	return e.EncodeElement(*n.Value, start)
}

func (n *NullableCodeAttributes) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, a := range start.Attr {
		if a.Name.Space == "http://www.w3.org/2001/XMLSchema-instance" &&
			a.Name.Local == "nil" &&
			a.Value == "true" {

			n.Nil = true

			// consume element
			var discard any
			return d.DecodeElement(&discard, &start)
		}
	}

	var v CodeAttributes
	if err := d.DecodeElement(&v, &start); err != nil {
		return err
	}

	n.Value = &v
	return nil
}
