// Package metadata_models: file contains go representation of xsd structure: https://github.com/imi-bigpicture/bigpicture-metaflex/blob/2.0.0/src/BP.policy.xsd
package metadata_models

import (
	"encoding/xml"
)

type PolicySet struct {
	XMLName  xml.Name `xml:"POLICY_SET"`
	Policies []Policy `xml:"POLICY"`
}

type Policy struct {
	ObjectType
	DatasetRef Reference          `xml:"DATASET_REF"`
	Attributes NullableAttributes `xml:"ATTRIBUTES"`
}

func (ps *PolicySet) Equal(compare *PolicySet) bool {
	if compare == nil && ps != nil {
		return false
	}

	if compare != nil && ps == nil {
		return false
	}

	if compare == nil {
		return true
	}

	if len(ps.Policies) != len(compare.Policies) {
		return false
	}

	for i, policy := range ps.Policies {
		if !policy.Equal(&compare.Policies[i]) {
			return false
		}
	}

	return true
}

// Equal does not compare the Policy accession or alias, and datasetRef accession or alias
func (p *Policy) Equal(compare *Policy) bool {
	if compare == nil && p != nil {
		return false
	}

	if compare != nil && p == nil {
		return false
	}

	if compare == nil {
		// Both are nil
		return true
	}

	return p.Attributes.Equal(&compare.Attributes)
}
