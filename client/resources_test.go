package client

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/resources"
)

type resourcesFixture struct {
	t     *testing.T
	data  string
	cache string
}

func newResourcesFixture(t *testing.T) *resourcesFixture {
	t.Helper()
	root := t.TempDir()
	f := &resourcesFixture{t: t, data: filepath.Join(root, "Data"), cache: filepath.Join(root, "packs")}
	for _, directory := range []string{f.data, f.cache} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return f
}

func (f *resourcesFixture) write(root, relative, content string) string {
	f.t.Helper()
	file := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		f.t.Fatal(err)
	}
	return file
}

func (f *resourcesFixture) packed(root, relative, content string) string {
	f.t.Helper()
	encoded, err := resources.EncodeDVPL([]byte(content), resources.DVPL(1))
	if err != nil {
		f.t.Fatal(err)
	}
	return f.write(root, relative, string(encoded))
}

func (f *resourcesFixture) open(caseFold bool) *Resources {
	f.t.Helper()
	resources, err := openResources(f.data, f.cache, caseFold)
	if err != nil {
		f.t.Fatal(err)
	}
	return resources
}

func TestResourcesCacheOverlayWinsAndPlainBeatsDVPL(t *testing.T) {
	f := newResourcesFixture(t)
	f.write(f.data, "Gfx/icon.png", "data icon")
	f.write(f.cache, "Gfx/icon.png", "cache icon")
	f.write(f.data, "XML/config.xml", "plain config")
	f.packed(f.data, "XML/config.xml.dvpl", "packed config")
	f.packed(f.data, "Strings/ru.yaml.dvpl", "packed strings")

	r := f.open(false)
	for relative, want := range map[string]string{
		"Gfx/icon.png":        "cache icon",
		"XML/config.xml":      "plain config",
		"XML/config.xml.dvpl": "plain config",
		"Strings/ru.yaml":     "packed strings",
	} {
		got, err := r.Open(relative)
		if err != nil {
			t.Fatalf("%s: %v", relative, err)
		}
		if string(got) != want {
			t.Fatalf("%s: got %q, want %q", relative, got, want)
		}
	}
	if !r.Exists("Gfx/icon.png") || !r.Exists("Strings/ru.yaml") || r.Exists("missing.txt") {
		t.Fatal("wrong existence result")
	}
	path, ok := r.Path("Gfx/icon.png")
	if !ok || !strings.HasPrefix(path, f.cache) {
		t.Fatalf("cache path not selected: %q, %v", path, ok)
	}
}

func TestResourcesRejectsPathsEscapingRoots(t *testing.T) {
	f := newResourcesFixture(t)
	outside := f.write(filepath.Dir(f.data), "outside.txt", "outside")
	f.write(f.data, "inside.txt", "inside")
	r := f.open(false)
	for _, relative := range []string{"../outside.txt", "..\\outside.txt", "/etc/passwd", `C:\Windows\system.ini`, "a/../../outside.txt"} {
		if _, err := r.Open(relative); blitz.AsError(err).Code != "CLIENT_RESOURCE_INVALID_PATH" {
			t.Fatalf("%s: %v", relative, err)
		}
		if r.Exists(relative) {
			t.Fatalf("%s exists", relative)
		}
	}
	if data, err := os.ReadFile(outside); err != nil || string(data) != "outside" {
		t.Fatalf("outside file changed: %q, %v", data, err)
	}
}

func TestResourcesListMergesCacheAndSorts(t *testing.T) {
	f := newResourcesFixture(t)
	f.write(f.data, "Gfx/A/two.txt", "two")
	f.write(f.data, "Gfx/A/one.png", "data one")
	f.write(f.data, "Gfx/AB/three.txt", "three")
	f.write(f.data, "XML/x.yaml", "x")
	f.write(f.cache, "Gfx/A/one.png", "cache one")
	f.write(f.cache, "Gfx/C/four.txt", "four")
	r := f.open(false)

	for prefix, want := range map[string][]string{
		"":        {"Gfx/A/one.png", "Gfx/A/two.txt", "Gfx/AB/three.txt", "Gfx/C/four.txt", "XML/x.yaml"},
		"Gfx":     {"Gfx/A/one.png", "Gfx/A/two.txt", "Gfx/AB/three.txt", "Gfx/C/four.txt"},
		"Gfx/A":   {"Gfx/A/one.png", "Gfx/A/two.txt"},
		"Gfx/AB":  {"Gfx/AB/three.txt"},
		"Missing": {},
	} {
		got, err := r.List(prefix)
		if err != nil {
			t.Fatalf("%q: %v", prefix, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("List(%q) = %v, want %v", prefix, got, want)
		}
	}
	if data, err := r.Open("Gfx/A/one.png"); err != nil || string(data) != "cache one" {
		t.Fatalf("cache did not win in listing: %q, %v", data, err)
	}
}

func TestResourcesCaseFoldFindsAuditTextureScenario(t *testing.T) {
	f := newResourcesFixture(t)
	// The audit scenario: the client ships a differently cased texture than
	// the material references. On Windows and macOS the lookup must succeed.
	f.packed(f.data, "images/Hull.dx11.dds.dvpl", "hull texture")
	r := f.open(true)
	if !r.Exists("images/hull.dx11.dds") {
		t.Fatal("differently cased resource not found")
	}
	data, err := r.Open("images/hull.dx11.dds")
	if err != nil || string(data) != "hull texture" {
		t.Fatalf("got %q, %v", data, err)
	}
	exact := f.open(false)
	if exact.Exists("images/hull.dx11.dds") {
		t.Fatal("case-sensitive lookup matched a differently cased path")
	}
}

func TestResourcesCaseFoldDetectsCollisions(t *testing.T) {
	f := newResourcesFixture(t)
	f.write(f.data, "Images/Foo.txt", "upper")
	f.write(f.data, "Images/foo.txt", "lower")
	if _, err := openResources(f.data, f.cache, true); blitz.AsError(err).Code != "CLIENT_RESOURCE_COLLISION" {
		t.Fatalf("expected collision, got %v", err)
	}
	if _, err := openResources(f.data, f.cache, false); err != nil {
		t.Fatalf("case-sensitive view rejected distinct names: %v", err)
	}

	// A cache entry replaces a differently cased Data entry instead of
	// colliding with it.
	other := newResourcesFixture(t)
	other.write(other.data, "Images/Foo.txt", "data")
	other.write(other.cache, "images/foo.txt", "cache")
	r := other.open(true)
	data, err := r.Open("IMAGES/FOO.TXT")
	if err != nil || string(data) != "cache" {
		t.Fatalf("got %q, %v", data, err)
	}
}

func TestResourcesCaseFoldTreatsPlainAndDVPLSiblingAsOne(t *testing.T) {
	f := newResourcesFixture(t)
	f.write(f.data, "Images/Foo.dds", "plain")
	f.packed(f.data, "images/foo.DDS.dvpl", "packed")
	r := f.open(true)
	data, err := r.Open("IMAGES/FOO.DDS")
	if err != nil || string(data) != "plain" {
		t.Fatalf("got %q, %v", data, err)
	}
}

func TestLogicalResourceKeyNormalizesAndFolds(t *testing.T) {
	for _, test := range []struct {
		input string
		fold  bool
		want  string
		ok    bool
	}{
		{"Gfx/A/../A/Icon.PNG", false, "Gfx/A/Icon.PNG", true},
		{"Gfx\\A\\Icon.PNG", true, "gfx/a/icon.png", true},
		{"Strings/ru.yaml.dvpl", false, "Strings/ru.yaml", true},
		{"Strings/ru.yaml.DVPL", true, "strings/ru.yaml", true},
		{"../outside", false, "", false},
		{"/absolute", false, "", false},
		{`C:\absolute`, false, "", false},
		{"", false, "", false},
		{".", false, "", false},
	} {
		got, ok := LogicalResourceKey(test.input, test.fold)
		if got != test.want || ok != test.ok {
			t.Fatalf("LogicalResourceKey(%q, %v) = %q, %v; want %q, %v", test.input, test.fold, got, ok, test.want, test.ok)
		}
	}
}

func TestOpenResourcesRequiresDataDirectory(t *testing.T) {
	if _, err := OpenResources(filepath.Join(t.TempDir(), "missing"), ""); blitz.AsError(err).Code != "IO_ERROR" {
		t.Fatalf("expected IO_ERROR, got %v", err)
	}
}
