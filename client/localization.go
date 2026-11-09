package client

import (
	blitz "github.com/thecoooler/blitz-client"
	"go.yaml.in/yaml/v3"
)

// Localization is a client localization document. A nil value is an explicit
// null entry, which is not the same as an absent key.
type Localization map[string]*string

// Resolved returns the non-null translations. The catalog uses these strings
// where export preserves the null entries.
func (localization Localization) Resolved() map[string]string {
	resolved := make(map[string]string, len(localization))
	for key, value := range localization {
		if value != nil {
			resolved[key] = *value
		}
	}
	return resolved
}

// DecodeLocalization parses a Strings/ru.yaml document without dropping nulls.
func DecodeLocalization(data []byte) (Localization, error) {
	var entries map[string]*string
	if err := yaml.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return Localization(entries), nil
}

// ReadLocalization reads Strings/ru.yaml. A missing resource yields an empty
// document, like the catalog's optional localization.
func ReadLocalization(source CatalogSource) (Localization, error) {
	const relative = "Strings/ru.yaml"
	if !source.Exists(relative) {
		return Localization{}, nil
	}
	data, err := source.Open(relative)
	if err != nil {
		return nil, err
	}
	localization, err := DecodeLocalization(data)
	if err != nil {
		return nil, blitz.ErrorAt("CLIENT_YAML_INVALID", err.Error(), source.Location(relative))
	}
	return localization, nil
}

// readLocalizedNames is the catalog's resolved-string projection.
func readLocalizedNames(dataDir string) (map[string]string, error) {
	localization, err := ReadLocalization(directorySource{dir: dataDir})
	if err != nil {
		return nil, err
	}
	return localization.Resolved(), nil
}
