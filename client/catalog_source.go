package client

import (
	"fmt"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/resources/xml"
	"go.yaml.in/yaml/v3"
)

// CatalogSource supplies catalog source documents. Paths are relative to the
// client's Data directory, using forward slashes. Location names a resource for
// diagnostics and may be the physical file behind an overlay.
type CatalogSource interface {
	Exists(relative string) bool
	Open(relative string) ([]byte, error)
	Location(relative string) string
}

// directorySource reads catalog documents directly from a Data directory.
type directorySource struct {
	dir string
}

func (s directorySource) Exists(relative string) bool {
	return ResolveClientAsset(s.dir, relative) != ""
}

func (s directorySource) Open(relative string) ([]byte, error) {
	path := ResolveClientAsset(s.dir, relative)
	if path == "" {
		return nil, blitz.ErrorAt("CLIENT_DATA_MISSING", "required client resource is missing", relative)
	}
	return ReadClientAsset(path)
}

func (s directorySource) Location(relative string) string {
	if path := ResolveClientAsset(s.dir, relative); path != "" {
		return path
	}
	return relative
}

// ReadAvailableNations reads available_nations.yaml, falling back to the
// built-in nation order when the resource is absent. A present document with
// no entries is invalid.
func ReadAvailableNations(source CatalogSource) ([]string, error) {
	const relative = "available_nations.yaml"
	if !source.Exists(relative) {
		result := make([]string, len(nations))
		for index, nation := range nations {
			result[index] = nation.ID
		}
		return result, nil
	}
	data, err := source.Open(relative)
	if err != nil {
		return nil, err
	}
	var document struct {
		Nations []string `yaml:"available_nations"`
	}
	if err := yaml.Unmarshal(data, &document); err != nil {
		return nil, blitz.ErrorAt("CLIENT_YAML_INVALID", err.Error(), source.Location(relative))
	}
	if len(document.Nations) == 0 {
		return nil, blitz.ErrorAt("CLIENT_YAML_INVALID",
			"available_nations.yaml contains no nation entries", source.Location(relative))
	}
	return document.Nations, nil
}

// VehicleList is one nation's parsed XML/item_defs/vehicles/<nation>/list.xml.
// Path is the source location used in diagnostics.
type VehicleList struct {
	Nation  string
	Path    string
	Entries []VehicleListEntry
}

// VehicleListEntry is one vehicle node of a list. Node keeps all source fields
// for callers that project complete metadata. The scalar fields are the values
// catalog discovery and export share; HasLevel reports whether the text is
// nonempty.
type VehicleListEntry struct {
	TechName   string
	Node       *xml.Node
	ID         string
	UserString string
	Level      string
	HasLevel   bool
}

// ReadVehicleList parses one nation's vehicle list. A missing list returns
// (nil, nil). Duplicate ids and technical names within the list are invalid.
// The nextAvailableId list metadata node is not a vehicle and is skipped.
func ReadVehicleList(source CatalogSource, nation string) (*VehicleList, error) {
	relative := "XML/item_defs/vehicles/" + nation + "/list.xml"
	if !source.Exists(relative) {
		return nil, nil
	}
	data, err := source.Open(relative)
	if err != nil {
		return nil, err
	}
	root, err := xml.Parse(data)
	if err != nil {
		return nil, blitz.ErrorAt("CLIENT_XML_INVALID", err.Error(), source.Location(relative))
	}
	list := &VehicleList{Nation: nation, Path: source.Location(relative), Entries: []VehicleListEntry{}}
	ids := map[string]bool{}
	names := map[string]bool{}
	for _, node := range root.Children {
		if node.Name == "nextAvailableId" {
			continue
		}
		if names[node.Name] {
			return nil, blitz.ErrorAt("CLIENT_XML_INVALID",
				fmt.Sprintf("duplicate vehicle technical name %s:%s", nation, node.Name), list.Path)
		}
		names[node.Name] = true
		entry := VehicleListEntry{TechName: node.Name, Node: node}
		entry.ID, _ = xml.ChildText(node, "id")
		entry.UserString, _ = xml.ChildText(node, "userString")
		entry.Level, entry.HasLevel = xml.ChildText(node, "level")
		if entry.ID != "" {
			if ids[entry.ID] {
				return nil, blitz.ErrorAt("CLIENT_XML_INVALID",
					fmt.Sprintf("duplicate vehicle id %s:%s", nation, entry.ID), list.Path)
			}
			ids[entry.ID] = true
		}
		list.Entries = append(list.Entries, entry)
	}
	return list, nil
}
