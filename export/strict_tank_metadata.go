package export

import (
	"encoding/json"
	"fmt"
)

// StrictTankMetadata describes the list.xml fields observed in the current game
// resources. XML scalars remain strings: the exporter does not interpret their
// numeric or boolean meaning. Optional distinguishes absent, null and present.
// Unknown fields are rejected, so newer resources may require a schema update.
type StrictTankMetadata struct {
	CombatRole                  Optional[map[string]Optional[string]] `json:"combatRole,omitzero"`
	ConfigurationModes          Optional[string]                      `json:"configurationModes,omitzero"`
	Description                 Optional[XMLLocalizationKey]          `json:"description,omitzero"`
	EnrichmentPermanentCost     Optional[XMLInteger]                  `json:"enrichmentPermanentCost,omitzero"`
	Level                       Optional[XMLInteger]                  `json:"level,omitzero"`
	Price                       Optional[StrictMetadataPrice]         `json:"price,omitzero"`
	SellPrice                   Optional[StrictMetadataPrice]         `json:"sellPrice,omitzero"`
	Tags                        Optional[string]                      `json:"tags,omitzero"`
	UserString                  Optional[XMLLocalizationKey]          `json:"userString,omitzero"`
	ShortUserString             Optional[XMLLocalizationKey]          `json:"shortUserString,omitzero"`
	NotInShop                   Optional[XMLBool]                     `json:"notInShop,omitzero"`
	UseSellPriceForCompensation Optional[XMLBool]                     `json:"useSellPriceForCompensation,omitzero"`
	Leasing                     Optional[string]                      `json:"leasing,omitzero"`
}

// StrictMetadataPrice is either an XML scalar or an object with XML text and
// gold children. Exactly one of Scalar or Details must be non-nil.
type StrictMetadataPrice struct {
	Scalar  *XMLInteger
	Details *StrictMetadataPriceDetails
}

type StrictMetadataPriceDetails struct {
	Text Optional[XMLInteger]              `json:"#text,omitzero"`
	Gold Optional[StrictMetadataGoldPrice] `json:"gold,omitzero"`
}

type StrictMetadataGoldPrice struct {
	DateFactors Optional[StrictMetadataDateFactors] `json:"dateFactors,omitzero"`
}

type StrictMetadataDateFactors struct {
	Factor Optional[StrictMetadataDateFactor] `json:"factor,omitzero"`
}

type StrictMetadataDateFactor struct {
	ExpirationDate Optional[string]     `json:"expirationDate,omitzero"`
	Value          Optional[XMLInteger] `json:"value,omitzero"`
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
