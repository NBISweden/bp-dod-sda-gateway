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

	// Not comparing Policy accession or alias
	//if p.Accession != compare.Accession {
	//	return false
	//}
	//if p.Alias != compare.Alias {
	//	return false
	//}

	// Not comparing datasetRef accession or alias
	//if p.DatasetRef.Alias != compare.DatasetRef.Alias {
	//	return false
	//}
	//if p.DatasetRef.Accession != compare.DatasetRef.Accession {
	//	return false
	//}

	return p.Attributes.Equal(&compare.Attributes)
}
