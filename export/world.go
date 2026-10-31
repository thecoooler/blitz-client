package export

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/thecoooler/blitz-client/client"
	"github.com/thecoooler/blitz-client/resources/sprite"
)

// WorldCatalog indexes map records and mechanics without duplicating source data.
// Paths to media are relative to the export directory.
type WorldCatalog struct {
	Maps           []WorldMap        `json:"maps"`
	Mechanics      []WorldMechanic   `json:"mechanics"`
	MechanicsMedia []WorldMedia      `json:"mechanicsMedia,omitempty"`
	MediaIssues    []WorldMediaIssue `json:"mediaIssues,omitempty"`
}
type WorldMap struct {
	Key            string       `json:"key"`
	NameKey        string       `json:"nameKey,omitempty"`
	DescriptionKey string       `json:"descriptionKey,omitempty"`
	TacticsKey     string       `json:"tacticsKey,omitempty"`
	Images         []WorldMedia `json:"images,omitempty"`
}
type WorldMechanic struct {
	ID             string   `json:"id"`
	NameKey        string   `json:"nameKey,omitempty"`
	DescriptionKey string   `json:"descriptionKey,omitempty"`
	Tanks          []string `json:"tanks"`
}
type WorldMedia struct {
	Source string `json:"source"`
	Frame  int    `json:"frame"`
	File   string `json:"file"`
}
type worldWriteError struct{ err error }

func (e *worldWriteError) Error() string { return e.err.Error() }
func (e *worldWriteError) Unwrap() error { return e.err }

type WorldMediaIssue struct {
	Source  string `json:"source"`
	Message string `json:"message"`
}

// buildWorldCatalog records only relationships supported by the source files.
func buildWorldCatalog(s *Snapshot) *WorldCatalog {
	c := &WorldCatalog{Maps: []WorldMap{}, Mechanics: []WorldMechanic{}}
	ru, _ := s.Localization["ru"].(map[string]any)
	has := func(k string) bool { _, ok := ru[k]; return ok }
	if root, ok := s.Maps.(map[string]any); ok {
		if maps, ok := root["maps"].(map[string]any); ok {
			keys := make([]string, 0, len(maps))
			for k := range maps {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				m := WorldMap{Key: k}
				fields, _ := maps[k].(map[string]any)
				if local, ok := fields["localName"].(string); ok {
					key := "#maps:" + k + ":" + local
					if has(key) {
						m.NameKey = key
					}
				}
				if key := "Maps/Description/" + k; has(key) {
					m.DescriptionKey = key
				}
				if key := "Maps/Tactics/" + k; has(key) {
					m.TacticsKey = key
				}
				c.Maps = append(c.Maps, m)
			}
		}
	}
	refs := map[string][]string{}
	for _, tank := range s.Tanks {
		p, _ := tank.Parameters.(map[string]any)
		list, _ := p["mechanics"].([]any)
		seen := map[string]bool{}
		for _, item := range list {
			m, _ := item.(map[string]any)
			id, _ := m["name"].(string)
			if id != "" && !seen[id] {
				refs[id] = append(refs[id], tank.ID)
				seen[id] = true
			}
		}
	}
	// Include unassigned localized mechanics as well as those attached to tanks.
	for key := range ru {
		if id, ok := strings.CutPrefix(key, "TankMechanics/"); ok && !strings.HasSuffix(id, "Description") && id != "" {
			if _, exists := refs[id]; !exists {
				refs[id] = []string{}
			}
		}
	}
	ids := make([]string, 0, len(refs))
	for id := range refs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		m := WorldMechanic{ID: id, Tanks: refs[id]}
		if key := "TankMechanics/" + id; has(key) {
			m.NameKey = key
		}
		if key := "TankMechanics/" + id + "Description"; has(key) {
			m.DescriptionKey = key
		}
		c.Mechanics = append(c.Mechanics, m)
	}
	return c
}

// exportWorldMedia renders installed map previews and mechanics artwork. Files
// from the packs overlay are resolved independently for descriptors and atlases.
// Battle HUD assets are visual elements of mechanics, not their lobby icons.
func exportWorldMedia(c *WorldCatalog, files map[string]sourceFile, out string) error {
	const mapDir = "Gfx/Lobby/backgrounds/premium/maps/"
	const mechanicDir = "Gfx/Lobby/icons/tank-mechanics/"
	candidates := map[string]string{}
	for name := range files {
		rel := filepath.ToSlash(name)
		if !strings.HasPrefix(rel, mapDir) && !strings.HasPrefix(rel, mechanicDir) && !mechanicsHUDResource(rel) {
			continue
		}
		stem, _, ok := iconFileStem(rel)
		if !ok {
			continue
		}
		base := strings.TrimSuffix(stem, "@2x")
		// Prefer high resolution, then a descriptor over an image of the same name.
		old := candidates[base]
		rank := func(s string) int {
			if s == "" {
				return -1
			}
			n := 0
			x, kind, _ := iconFileStem(s)
			if strings.HasSuffix(x, "@2x") {
				n += 2
			}
			if kind == "sprite" {
				n++
			}
			return n
		}
		if rank(rel) > rank(old) || (rank(rel) == rank(old) && rel < old) {
			candidates[base] = rel
		}
	}
	keys := make([]string, 0, len(candidates))
	for k := range candidates {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	maps := map[string]int{}
	for i, m := range c.Maps {
		maps[m.Key] = i
	}
	for _, base := range keys {
		rel := candidates[base]
		// Only attach map previews whose resource name matches a map key.
		idx, matched := 0, false
		if strings.HasPrefix(base, mapDir) {
			id := strings.TrimPrefix(base, mapDir)
			idx, matched = maps[id]
			if !matched {
				idx, matched = maps[strings.TrimSuffix(id, "_mini")]
			}
			if !matched {
				continue
			}
		}
		media, err := renderWorldResource(files, rel, out)
		if err != nil {
			var writeErr *worldWriteError
			if errors.As(err, &writeErr) {
				return writeErr
			}
			c.MediaIssues = append(c.MediaIssues, WorldMediaIssue{Source: rel, Message: err.Error()})
			continue
		}
		if matched {
			c.Maps[idx].Images = append(c.Maps[idx].Images, media...)
		} else {
			c.MechanicsMedia = append(c.MechanicsMedia, media...)
		}
	}
	return nil
}

// Mechanics-specific HUD paths are referenced by the corresponding battle UI.
// The shared sight and the Polish consumable UI are deliberately excluded.
func mechanicsHUDResource(rel string) bool {
	const aim = "Gfx/Battle/HUD/aim/"
	for _, dir := range []string{"ShotDispersionStabilizator/", "twin-guns/", "gun-cooling/", "atgm/", "missile/"} {
		if strings.HasPrefix(rel, aim+dir) {
			return true
		}
	}
	stem, _, ok := iconFileStem(rel)
	if !ok {
		return false
	}
	stem = strings.TrimSuffix(stem, "@2x")
	return stem == aim+"PumpDrum" || stem == aim+"PumpDrumRecharge"
}

func renderWorldResource(files map[string]sourceFile, rel, out string) ([]WorldMedia, error) {
	source, ok := files[filepath.FromSlash(rel)]
	if !ok {
		return nil, fmt.Errorf("resource missing")
	}
	data, err := client.ReadClientAsset(source.path)
	if err != nil {
		return nil, err
	}
	var desc *sprite.Sprite
	if strings.HasSuffix(rel, ".txt") {
		desc, err = sprite.Parse(data)
	} else {
		desc, _, err = sprite.Embedded(data)
	}
	if err != nil {
		return nil, err
	}
	var frames []image.Image
	if desc == nil {
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		frames = append(frames, img)
	} else {
		atlases := map[string]image.Image{}
		for _, frame := range desc.Frames {
			texture := rel
			if name := desc.Textures[frame.Texture]; name != "" {
				texture, err = iconTexturePath(rel, name)
				if err != nil {
					return nil, err
				}
			}
			img := atlases[texture]
			if img == nil {
				f, ok := files[filepath.FromSlash(texture)]
				if !ok {
					return nil, fmt.Errorf("missing sprite atlas %s", texture)
				}
				raw, err := client.ReadClientAsset(f.path)
				if err != nil {
					return nil, err
				}
				img, _, err = image.Decode(bytes.NewReader(raw))
				if err != nil {
					return nil, err
				}
				atlases[texture] = img
			}
			if frame.X+frame.Width > img.Bounds().Dx() || frame.Y+frame.Height > img.Bounds().Dy() {
				return nil, fmt.Errorf("sprite frame outside atlas %s", texture)
			}
			canvas := image.NewNRGBA(image.Rect(0, 0, desc.Width, desc.Height))
			target := image.Rect(frame.OffsetX, frame.OffsetY, frame.OffsetX+frame.Width, frame.OffsetY+frame.Height)
			draw.Draw(canvas, target, img, img.Bounds().Min.Add(image.Pt(frame.X, frame.Y)), draw.Src)
			frames = append(frames, canvas)
		}
	}
	stem, _, _ := iconFileStem(rel)
	stem = strings.TrimSuffix(stem, "@2x")
	if !safeDataPath(stem) {
		return nil, fmt.Errorf("unsafe media path %q", stem)
	}
	result := make([]WorldMedia, 0, len(frames))
	for i, img := range frames {
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return nil, err
		}
		suffix := ""
		if i > 0 {
			suffix = fmt.Sprintf("-frame%d", i)
		}
		dest := path.Join("assets/world", stem+suffix+".png")
		if err := writeDataFile(out, dest, buf.Bytes()); err != nil {
			return nil, &worldWriteError{err}
		}
		result = append(result, WorldMedia{Source: rel, Frame: i, File: dest})
	}
	return result, nil
}
