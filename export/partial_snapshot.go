package export

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// PartialSnapshot is a v1 export in which every data section is decoded with
// the strict schema, while sections without a schema stay raw: tank icons and
// model media, crewSkillTraining and unknown settings documents. Raw parts are
// preserved but not validated; UnvalidatedSections lists those present in this
// document. DecodeStrictDataExport rejects any document that has them.
//
// The typed schemas describe the currently observed data and may reject new
// fields or shapes in later game versions.
type PartialSnapshot struct {
	Format              string                                                `json:"format"`
	Version             int                                                   `json:"version"`
	Included            []string                                              `json:"included"`
	Game                Optional[SnapshotGame]                                `json:"game,omitzero"`
	ExportedAt          Optional[string]                                      `json:"exportedAt,omitzero"`
	Nations             []string                                              `json:"nations"`
	Tanks               []PartialSnapshotTank                                 `json:"tanks"`
	Localization        Optional[StrictLocalization]                          `json:"localization,omitzero"`
	Maps                Optional[StrictMaps]                                  `json:"maps,omitzero"`
	Components          Optional[map[string]Optional[StrictNationComponents]] `json:"components,omitzero"`
	Equipment           Optional[StrictEquipment]                             `json:"equipment,omitzero"`
	Provisions          Optional[StrictProvisions]                            `json:"provisions,omitzero"`
	Settings            Optional[PartialSettings]                             `json:"settings,omitzero"`
	HangarParameters    Optional[StrictHangarParameters]                      `json:"hangarParameters,omitzero"`
	CrewSkills          Optional[[]StrictCrewSkill]                           `json:"crewSkills,omitzero"`
	CrewSkillTraining   Optional[json.RawMessage]                             `json:"crewSkillTraining,omitzero"`
	Issues              []StrictExportIssue                                   `json:"issues"`
	UnvalidatedSections []string                                              `json:"-"`
}

// PartialSnapshotTank keeps media that lack a schema as raw JSON; numbers and
// configuration array order are retained.
type PartialSnapshotTank struct {
	ID            string                            `json:"id"`
	Nation        string                            `json:"nation"`
	ClientID      int64                             `json:"clientId"`
	TechName      string                            `json:"techName"`
	Metadata      Optional[StrictTankMetadata]      `json:"metadata,omitzero"`
	Parameters    Optional[StrictTankParameters]    `json:"parameters,omitzero"`
	Configuration Optional[StrictTankConfiguration] `json:"configuration,omitzero"`
	Icons         Optional[json.RawMessage]         `json:"icons,omitzero"`
	Model         Optional[json.RawMessage]         `json:"model,omitzero"`
}

// ReadPartialClientExport reads data.json in directory with
// DecodePartialClientExport.
func ReadPartialClientExport(directory string) (*PartialSnapshot, error) {
	s, _, err := readPartialClientExport(directory, schemaOptions{})
	return s, err
}

// ReadPartialClientExportLenient reads data.json in directory with
// DecodePartialClientExportLenient.
func ReadPartialClientExportLenient(directory string) (*PartialSnapshot, []SchemaWarning, error) {
	return readPartialClientExport(directory, schemaOptions{lenient: true})
}

func readPartialClientExport(directory string, options schemaOptions) (*PartialSnapshot, []SchemaWarning, error) {
	path := filepath.Join(directory, "data.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	snapshot, warnings, err := decodePartialClientExport(data, options)
	if err != nil {
		return nil, warnings, fmt.Errorf("%s: %w", path, err)
	}
	return snapshot, warnings, nil
}

// DecodePartialClientExport reads a single v1 JSON document of any --include
// selection. Every data section is validated with the strict schema; media and
// crewSkillTraining stay raw and are listed in UnvalidatedSections. Numbers
// retain their spelling, absent and null fields stay distinct, and arrays keep
// their order. Errors are *SchemaError values with the JSON path of the
// offending value, or JSON syntax errors.
func DecodePartialClientExport(r io.Reader) (*PartialSnapshot, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	s, _, err := decodePartialClientExport(data, schemaOptions{})
	return s, err
}

// DecodePartialClientExportLenient is DecodePartialClientExport for data that
// may come from a newer game version. Unknown fields are skipped, and a value
// with an unexpected shape or scalar spelling is dropped (the field becomes
// absent, the dictionary entry or array element is removed); each case is
// reported as a SchemaWarning. Malformed JSON, duplicate keys, an unsupported
// format version and a missing or invalid root field are still errors.
func DecodePartialClientExportLenient(r io.Reader) (*PartialSnapshot, []SchemaWarning, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}
	return decodePartialClientExport(data, schemaOptions{lenient: true})
}

func decodePartialClientExport(data []byte, options schemaOptions) (*PartialSnapshot, []SchemaWarning, error) {
	var s PartialSnapshot
	warnings, err := decodeSchemaJSON(data, &s, options)
	if err != nil {
		return nil, warnings, err
	}
	if s.Format != "blitz-client" || s.Version != 1 {
		return nil, warnings, fmt.Errorf("unsupported client export %q version %d", s.Format, s.Version)
	}
	if len(s.Included) == 0 {
		return nil, warnings, fmt.Errorf("snapshot is missing included sections")
	}
	if _, err := ParseExportIncludes(s.Included); err != nil {
		return nil, warnings, err
	}
	if s.CrewSkillTraining.Present() {
		s.UnvalidatedSections = append(s.UnvalidatedSections, "crewSkillTraining")
	}
	if settings, ok := s.Settings.Get(); ok {
		for _, name := range settings.UnvalidatedDocuments() {
			s.UnvalidatedSections = append(s.UnvalidatedSections, "settings."+name)
		}
	}
	for _, field := range []string{"icons", "model"} {
		for _, tank := range s.Tanks {
			if (field == "icons" && tank.Icons.Present()) || (field == "model" && tank.Model.Present()) {
				s.UnvalidatedSections = append(s.UnvalidatedSections, "tanks[]."+field)
				break
			}
		}
	}
	return &s, warnings, nil
}
