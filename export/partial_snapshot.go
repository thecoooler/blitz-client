package export

import (
	"bytes"
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
	Nations             []string                                              `json:"nations"`
	Tanks               []PartialSnapshotTank                                 `json:"tanks"`
	Localization        Optional[StrictLocalization]                          `json:"localization,omitempty"`
	Maps                Optional[StrictMaps]                                  `json:"maps,omitempty"`
	Components          Optional[map[string]Optional[StrictNationComponents]] `json:"components,omitempty"`
	Equipment           Optional[StrictEquipment]                             `json:"equipment,omitempty"`
	Provisions          Optional[StrictProvisions]                            `json:"provisions,omitempty"`
	Settings            Optional[PartialSettings]                             `json:"settings,omitempty"`
	HangarParameters    Optional[StrictHangarParameters]                      `json:"hangarParameters,omitempty"`
	CrewSkills          Optional[[]StrictCrewSkill]                           `json:"crewSkills,omitempty"`
	CrewSkillTraining   Optional[json.RawMessage]                             `json:"crewSkillTraining,omitempty"`
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
	Metadata      Optional[StrictTankMetadata]      `json:"metadata,omitempty"`
	Parameters    Optional[StrictTankParameters]    `json:"parameters,omitempty"`
	Configuration Optional[StrictTankConfiguration] `json:"configuration,omitempty"`
	Icons         Optional[json.RawMessage]         `json:"icons,omitempty"`
	Model         Optional[json.RawMessage]         `json:"model,omitempty"`
}

// ReadPartialClientExport reads data.json in directory with the partially typed
// schema. Use UnvalidatedSections to find the fields not checked by the schema.
func ReadPartialClientExport(directory string) (*PartialSnapshot, error) {
	path := filepath.Join(directory, "data.json")
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	snapshot, err := DecodePartialClientExport(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return snapshot, nil
}

// DecodePartialClientExport reads a single v1 JSON document. It checks for
// duplicate keys and trailing input everywhere, and validates the typed fields;
// it does not validate the structure or meaning of raw fields.
func DecodePartialClientExport(r io.Reader) (*PartialSnapshot, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var root map[string]json.RawMessage
	if err := decodeStrictJSON(data, &root); err != nil {
		return nil, err
	}
	if root == nil {
		return nil, fmt.Errorf("snapshot must be an object")
	}
	for _, field := range []string{"format", "version", "included", "nations", "tanks", "issues"} {
		value, ok := root[field]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, fmt.Errorf("required field %q is missing or null", field)
		}
	}
	// Validate array elements as well: encoding/json accepts null into strings
	// and structs, which would otherwise silently manufacture zero values.
	for _, field := range []string{"included", "nations", "tanks"} {
		var entries []json.RawMessage
		if err := json.Unmarshal(root[field], &entries); err != nil {
			return nil, fmt.Errorf("%s: %w", field, err)
		}
		if entries == nil {
			return nil, fmt.Errorf("%s must be an array", field)
		}
		for i, entry := range entries {
			if bytes.Equal(bytes.TrimSpace(entry), []byte("null")) {
				return nil, fmt.Errorf("%s[%d] cannot be null", field, i)
			}
			if field == "tanks" {
				if err := requirePartialTankIdentity(entry); err != nil {
					return nil, fmt.Errorf("tanks[%d]: %w", i, err)
				}
			}
		}
	}
	var s PartialSnapshot
	if err := decodeStrictJSON(data, &s); err != nil {
		return nil, err
	}
	if s.Format != "blitz-client" || s.Version != 1 {
		return nil, fmt.Errorf("unsupported client export %q version %d", s.Format, s.Version)
	}
	if len(s.Included) == 0 {
		return nil, fmt.Errorf("snapshot is missing included sections")
	}
	if _, err := ParseExportIncludes(s.Included); err != nil {
		return nil, err
	}
	if _, ok := root["crewSkillTraining"]; ok {
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
	return &s, nil
}

func requirePartialTankIdentity(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if fields == nil {
		return fmt.Errorf("expected object")
	}
	for _, field := range []string{"id", "nation", "clientId", "techName"} {
		value, ok := fields[field]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("required field %q is missing or null", field)
		}
	}
	return nil
}
