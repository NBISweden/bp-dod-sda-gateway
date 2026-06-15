package metadata_models

import (
	"encoding/xml"
	"slices"
	"strconv"
	"strings"
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

func (a *Attributes) Equal(compare *Attributes) bool {
	if compare == nil && a != nil {
		return false
	}

	if compare != nil && a == nil {
		return false
	}

	if compare == nil {
		// Both have nil value
		return true
	}

	if len(compare.StringAttributes) != len(a.StringAttributes) ||
		len(compare.NumericAttributes) != len(a.NumericAttributes) ||
		len(compare.MeasurementAttributes) != len(a.MeasurementAttributes) ||
		len(compare.CodeAttributes) != len(a.CodeAttributes) ||
		len(compare.SetAttributes) != len(a.SetAttributes) {
		return false
	}

	// Sort both to ensure same order
	slices.SortFunc(compare.StringAttributes, func(a, b StringAttribute) int {
		return strings.Compare(a.Tag, b.Tag)
	})
	slices.SortFunc(a.StringAttributes, func(a, b StringAttribute) int {
		return strings.Compare(a.Tag, b.Tag)
	})

	for i, sa := range a.StringAttributes {
		if !sa.Equal(&compare.StringAttributes[i]) {
			return false
		}
	}

	slices.SortFunc(compare.NumericAttributes, func(a, b NumericAttribute) int {
		return strings.Compare(a.Tag, b.Tag)
	})
	slices.SortFunc(a.NumericAttributes, func(a, b NumericAttribute) int {
		return strings.Compare(a.Tag, b.Tag)
	})

	for i, na := range a.NumericAttributes {
		if !na.Equal(&compare.NumericAttributes[i]) {
			return false
		}
	}

	slices.SortFunc(compare.MeasurementAttributes, func(a, b MeasurementAttribute) int {
		return strings.Compare(a.Tag, b.Tag)
	})
	slices.SortFunc(a.MeasurementAttributes, func(a, b MeasurementAttribute) int {
		return strings.Compare(a.Tag, b.Tag)
	})

	for i, ma := range a.MeasurementAttributes {
		if !ma.Equal(&compare.MeasurementAttributes[i]) {
			return false
		}
	}

	slices.SortFunc(compare.CodeAttributes, func(a, b CodeAttribute) int {
		return strings.Compare(a.Tag, b.Tag)
	})
	slices.SortFunc(a.CodeAttributes, func(a, b CodeAttribute) int {
		return strings.Compare(a.Tag, b.Tag)
	})

	for i, ca := range compare.CodeAttributes {
		if !ca.Equal(&compare.CodeAttributes[i]) {
			return false
		}
	}

	slices.SortFunc(compare.SetAttributes, func(a, b SetAttribute) int {
		return strings.Compare(a.Tag, b.Tag)
	})
	slices.SortFunc(a.SetAttributes, func(a, b SetAttribute) int {
		return strings.Compare(a.Tag, b.Tag)
	})

	for i, sa := range compare.SetAttributes {
		if !sa.Equal(&compare.SetAttributes[i]) {
			return false
		}
	}

	return true
}

type StringAttribute struct {
	Tag   string          `xml:"TAG"`
	Value *NullableString `xml:"VALUE"`
}

func (sa *StringAttribute) Equal(compare *StringAttribute) bool {
	if (compare == nil && sa != nil) ||
		(compare != nil && sa == nil) {
		return false
	}
	if compare == nil {
		// Both have nil value
		return true
	}

	if compare.Tag != sa.Tag {
		return false
	}

	return sa.Value.Equal(compare.Value)
}

type NumericAttribute struct {
	Tag   string         `xml:"TAG"`
	Value *NullableFloat `xml:"VALUE"`
}

func (na *NumericAttribute) Equal(compare *NumericAttribute) bool {
	if (compare == nil && na != nil) ||
		(compare != nil && na == nil) {
		return false
	}
	if compare == nil {
		// Both have nil value
		return true
	}

	if compare.Tag != na.Tag {
		return false
	}

	return na.Value.Equal(compare.Value)
}

type MeasurementAttribute struct {
	Tag   string         `xml:"TAG"`
	Value *NullableFloat `xml:"VALUE"`
	Units string         `xml:"UNITS"`
}

func (ma *MeasurementAttribute) Equal(compare *MeasurementAttribute) bool {
	if (compare == nil && ma != nil) ||
		(compare != nil && ma == nil) {
		return false
	}
	if compare == nil {
		// Both are nil
		return true
	}

	if compare.Units != ma.Units {
		return false
	}

	if ma.Tag != compare.Tag {
		return false
	}

	return ma.Value.Equal(compare.Value)
}

type CodeAttribute struct {
	Tag   string                      `xml:"TAG"`
	Value *NullableCodeAttributeValue `xml:"VALUE"`
}

func (ca *CodeAttribute) Equal(compare *CodeAttribute) bool {
	if (compare == nil && ca != nil) ||
		(compare != nil && ca == nil) {
		return false
	}
	if compare == nil {
		// Both are nil
		return true
	}
	if ca.Tag != compare.Tag {
		return false
	}

	return ca.Value.Equal(compare.Value)
}

type SetAttribute struct {
	Tag   string              `xml:"TAG"`
	Value *NullableAttributes `xml:"VALUE"`
}

func (sa *SetAttribute) Equal(compare *SetAttribute) bool {
	if (compare == nil && sa != nil) ||
		(compare != nil && sa == nil) {
		return false
	}
	if compare == nil {
		// Both are nil
		return true
	}

	if compare.Tag != sa.Tag {
		return false
	}

	return sa.Value.Equal(compare.Value)
}

type CodeAttributeValue struct {
	Code          string          `xml:"CODE"`
	Scheme        string          `xml:"SCHEME"`
	Meaning       string          `xml:"MEANING"`
	SchemeVersion *NullableString `xml:"SCHEME_VERSION"`
}

func (ca *CodeAttributeValue) Equal(compare *CodeAttributeValue) bool {
	if (compare == nil && ca != nil) ||
		(compare != nil && ca == nil) {
		return false
	}
	if compare == nil {
		// Both are nil
		return true
	}

	if ca.Code != compare.Code {
		return false
	}
	if ca.Scheme != compare.Scheme {
		return false
	}
	if ca.Meaning != compare.Meaning {
		return false
	}

	return ca.SchemeVersion.Equal(ca.SchemeVersion)
}

type CodeAttributes struct {
	CodeAttributes []CodeAttribute `xml:"CODE_ATTRIBUTE"`
}

type NullableString struct {
	Value *string
	Nil   bool
}

func (ns *NullableString) Equal(compare *NullableString) bool {
	if (compare == nil && ns != nil) ||
		(compare != nil && ns == nil) {
		return false
	}
	if compare == nil {
		// Both have nil value
		return true
	}

	if (compare.Value == nil && ns.Value != nil) ||
		(compare.Value != nil && ns.Value == nil) {
		return false
	}
	if compare.Value == nil {
		// Both have nil value
		return true
	}

	return &compare.Value == &ns.Value
}

func (ns *NullableString) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if ns.Nil {
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

	if ns.Value == nil {
		return e.EncodeElement(nil, start)
	}

	return e.EncodeElement(*ns.Value, start)
}

func (ns *NullableString) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, a := range start.Attr {
		if a.Name.Space == "http://www.w3.org/2001/XMLSchema-instance" &&
			a.Name.Local == "nil" &&
			a.Value == "true" {

			ns.Nil = true

			// consume element
			var discard any
			return d.DecodeElement(&discard, &start)
		}
	}

	var v string
	if err := d.DecodeElement(&v, &start); err != nil {
		return err
	}

	ns.Value = &v
	return nil
}

type NullableFloat struct {
	Value *float64
	Nil   bool
}

func (nf *NullableFloat) Equal(compare *NullableFloat) bool {
	if (compare == nil && nf != nil) ||
		(compare != nil && nf == nil) {
		return false
	}
	if compare == nil {
		// Both have nil value
		return true
	}

	if (compare.Value == nil && nf.Value != nil) ||
		(compare.Value != nil && nf.Value == nil) {
		return false
	}
	if compare.Value == nil {
		// Both have nil value
		return true
	}

	return &compare.Value == &nf.Value
}

func (nf *NullableFloat) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if nf.Nil {
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

	if nf.Value == nil {
		return e.EncodeElement(nil, start)
	}

	var s string
	if *nf.Value == 0 {
		s = "0.0"
	} else {
		s = strconv.FormatFloat(*nf.Value, 'g', -1, 64)
	}

	return e.EncodeElement(s, start)
}

func (nf *NullableFloat) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, a := range start.Attr {
		if a.Name.Space == "http://www.w3.org/2001/XMLSchema-instance" &&
			a.Name.Local == "nil" &&
			a.Value == "true" {

			nf.Nil = true

			// consume element
			var discard any
			return d.DecodeElement(&discard, &start)
		}
	}

	var v float64
	if err := d.DecodeElement(&v, &start); err != nil {
		return err
	}

	nf.Value = &v
	return nil
}

type NullableAttributes struct {
	Value *Attributes
	Nil   bool
}

func (na *NullableAttributes) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if na.Nil {
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

	if na.Value == nil {
		return e.EncodeElement(nil, start)
	}

	return e.EncodeElement(*na.Value, start)
}

func (na *NullableAttributes) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, a := range start.Attr {
		if a.Name.Space == "http://www.w3.org/2001/XMLSchema-instance" &&
			a.Name.Local == "nil" &&
			a.Value == "true" {

			na.Nil = true

			// consume element
			var discard any
			return d.DecodeElement(&discard, &start)
		}
	}

	var v Attributes
	if err := d.DecodeElement(&v, &start); err != nil {
		return err
	}

	na.Value = &v
	return nil
}

func (na *NullableAttributes) Equal(compare *NullableAttributes) bool {
	if (compare == nil && na != nil) ||
		(compare != nil && na == nil) {
		return false
	}
	if compare == nil {
		// Both have nil attributes
		return true
	}

	if (compare.Value == nil && na.Value != nil) ||
		(compare.Value != nil && na.Value == nil) {
		return false
	}
	if compare.Value == nil {
		// Both have nil attributes
		return true
	}

	return compare.Value.Equal(compare.Value)
}

type NullableCodeAttributeValue struct {
	Value *CodeAttributeValue
	Nil   bool
}

func (nca *NullableCodeAttributeValue) Equal(compare *NullableCodeAttributeValue) bool {
	if (compare == nil && nca != nil) ||
		(compare != nil && nca == nil) {
		return false
	}
	if compare == nil {
		// Both are nil
		return true
	}

	return nca.Value.Equal(compare.Value)
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
