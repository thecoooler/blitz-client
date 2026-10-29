package export

import (
	"fmt"
	"io"
	"strings"
)

// StrictDataSnapshot represents a v1 data-only export using typed values. It
// does not include media or server-config training, which the reader rejects.
// The schema describes observed wire shapes, not game semantics or references.
// Future game versions may require schema updates.
type StrictDataSnapshot struct {
	Format           string                                                `json:"format"`
	Version          int                                                   `json:"version"`
	Included         []string                                              `json:"included"`
	Nations          []string                                              `json:"nations"`
	Tanks            []StrictDataTank                                      `json:"tanks"`
	Localization     Optional[StrictLocalization]                          `json:"localization,omitempty"`
	Maps             Optional[StrictMaps]                                  `json:"maps,omitempty"`
	Components       Optional[map[string]Optional[StrictNationComponents]] `json:"components,omitempty"`
	Equipment        Optional[StrictEquipment]                             `json:"equipment,omitempty"`
	Provisions       Optional[StrictProvisions]                            `json:"provisions,omitempty"`
	Settings         Optional[StrictSettings]                              `json:"settings,omitempty"`
	HangarParameters Optional[StrictHangarParameters]                      `json:"hangarParameters,omitempty"`
	CrewSkills       Optional[[]StrictCrewSkill]                           `json:"crewSkills,omitempty"`
	Issues           []StrictExportIssue                                   `json:"issues"`
}

type StrictDataTank struct {
	ID            string                            `json:"id"`
	Nation        string                            `json:"nation"`
	ClientID      int64                             `json:"clientId"`
	TechName      string                            `json:"techName"`
	Metadata      Optional[StrictTankMetadata]      `json:"metadata,omitempty"`
	Parameters    Optional[StrictTankParameters]    `json:"parameters,omitempty"`
	Configuration Optional[StrictTankConfiguration] `json:"configuration,omitempty"`
}

// DecodeStrictDataExport reads a single data-only export. It rejects unknown
// settings documents, media attachments and server-config training because
// their contents are not covered by this schema. For those inputs use
// DecodePartialClientExport, which explicitly reports every unvalidated part.
// Numbers retain their original spelling; selection arrays retain their order.
func DecodeStrictDataExport(r io.Reader) (*StrictDataSnapshot, error) {
	s, err := DecodePartialClientExport(r)
	if err != nil {
		return nil, err
	}
	return requireFullyTypedData(s)
}

// ReadStrictDataExport reads data.json in directory using DecodeStrictDataExport.
func ReadStrictDataExport(directory string) (*StrictDataSnapshot, error) {
	s, err := ReadPartialClientExport(directory)
	if err != nil {
		return nil, err
	}
	return requireFullyTypedData(s)
}

func requireFullyTypedData(s *PartialSnapshot) (*StrictDataSnapshot, error) {
	if len(s.Included) != 1 || s.Included[0] != "data" {
		return nil, fmt.Errorf("strict data reader requires a data-only export")
	}
	if len(s.UnvalidatedSections) != 0 {
		return nil, fmt.Errorf("unvalidated sections: %s", strings.Join(s.UnvalidatedSections, ", "))
	}
	out := &StrictDataSnapshot{Format: s.Format, Version: s.Version, Included: s.Included, Nations: s.Nations,
		Localization: s.Localization, Maps: s.Maps, Components: s.Components, Equipment: s.Equipment, Provisions: s.Provisions,
		HangarParameters: s.HangarParameters, CrewSkills: s.CrewSkills, Issues: s.Issues, Tanks: make([]StrictDataTank, len(s.Tanks))}
	if s.Settings.Null() {
		out.Settings = NullValue[StrictSettings]()
	} else if settings, ok := s.Settings.Get(); ok {
		out.Settings = Some(strictSettingsFromPartial(settings))
	}
	for i, tank := range s.Tanks {
		out.Tanks[i] = StrictDataTank{ID: tank.ID, Nation: tank.Nation, ClientID: tank.ClientID, TechName: tank.TechName, Metadata: tank.Metadata, Parameters: tank.Parameters, Configuration: tank.Configuration}
	}
	return out, nil
}
