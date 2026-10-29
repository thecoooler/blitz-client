package export

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// StrictTankMetadata describes the list.xml fields observed in the current game
// resources. XML scalars remain strings: the exporter does not interpret their
// numeric or boolean meaning. Optional distinguishes absent, null and present.
// Unknown fields are rejected, so newer resources may require a schema update.
type StrictTankMetadata struct {
	CombatRole                  Optional[map[string]Optional[string]] `json:"combatRole,omitempty"`
	ConfigurationModes          Optional[string]                      `json:"configurationModes,omitempty"`
	Description                 Optional[string]                      `json:"description,omitempty"`
	EnrichmentPermanentCost     Optional[string]                      `json:"enrichmentPermanentCost,omitempty"`
	Level                       Optional[string]                      `json:"level,omitempty"`
	Price                       Optional[StrictMetadataPrice]         `json:"price,omitempty"`
	SellPrice                   Optional[StrictMetadataPrice]         `json:"sellPrice,omitempty"`
	Tags                        Optional[string]                      `json:"tags,omitempty"`
	UserString                  Optional[string]                      `json:"userString,omitempty"`
	ShortUserString             Optional[string]                      `json:"shortUserString,omitempty"`
	NotInShop                   Optional[string]                      `json:"notInShop,omitempty"`
	UseSellPriceForCompensation Optional[string]                      `json:"useSellPriceForCompensation,omitempty"`
	Leasing                     Optional[string]                      `json:"leasing,omitempty"`
}

func (m *StrictTankMetadata) UnmarshalJSON(data []byte) error {
	if len(bytes.TrimSpace(data)) == 0 || bytes.TrimSpace(data)[0] != '{' {
		return fmt.Errorf("tank metadata must be an object")
	}
	type fields StrictTankMetadata
	var result fields
	if err := decodeStrictJSON(data, &result); err != nil {
		return err
	}
	*m = StrictTankMetadata(result)
	return nil
}

// StrictMetadataPrice is either an XML scalar or an object with XML text and
// gold children. Exactly one of Scalar or Details must be non-nil.
type StrictMetadataPrice struct {
	Scalar  *string
	Details *StrictMetadataPriceDetails
}

type StrictMetadataPriceDetails struct {
	Text Optional[string]                  `json:"#text,omitempty"`
	Gold Optional[StrictMetadataGoldPrice] `json:"gold,omitempty"`
}

type StrictMetadataGoldPrice struct {
	DateFactors Optional[StrictMetadataDateFactors] `json:"dateFactors,omitempty"`
}

type StrictMetadataDateFactors struct {
	Factor Optional[StrictMetadataDateFactor] `json:"factor,omitempty"`
}

type StrictMetadataDateFactor struct {
	ExpirationDate Optional[string] `json:"expirationDate,omitempty"`
	Value          Optional[string] `json:"value,omitempty"`
}

func (p *StrictMetadataPrice) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return fmt.Errorf("empty metadata price")
	}
	switch data[0] {
	case '"':
		var value string
		if err := decodeStrictJSON(data, &value); err != nil {
			return err
		}
		*p = StrictMetadataPrice{Scalar: &value}
	case '{':
		var value StrictMetadataPriceDetails
		if err := decodeStrictJSON(data, &value); err != nil {
			return err
		}
		*p = StrictMetadataPrice{Details: &value}
	default:
		return fmt.Errorf("metadata price must be a string or object")
	}
	return nil
}

func (p StrictMetadataPrice) MarshalJSON() ([]byte, error) {
	if (p.Scalar == nil) == (p.Details == nil) {
		return nil, fmt.Errorf("metadata price must have exactly one variant")
	}
	if p.Scalar != nil {
		return json.Marshal(*p.Scalar)
	}
	return json.Marshal(p.Details)
}
