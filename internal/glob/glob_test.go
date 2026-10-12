package glob

import "testing"

func TestMatchesLikeRustGlobWithLiteralSeparators(t *testing.T) {
	cases := []struct {
		pattern, path string
		expected      bool
	}{
		{"**/*.yaml", "skip.yaml", true},
		{"**/*.yaml", "UI/battle.yaml", true},
		{"**/*.dds", "Gfx/a/tank.dds", true},
		{"*.yaml", "UI/battle.yaml", false},
		{"*.yaml", "twin.yaml", true},
		{"3d/**", "3d/tank.sc2", true},
		{"3d/**", "3d/a/b", true},
		{"3d/**", "3dx/a", false},
		{"Gfx/*.[w]eb?", "Gfx/image.webp", true},
		{"Gfx/*.[w]eb?", "Gfx/.hidden.webp", true},
		{"Gfx/*.[!w]eb?", "Gfx/image.webp", false},
		{"a/**/b", "a/b", true},
		{"a/**/b", "a/x/y/b", true},
		{"?", "/", false},
		{"[a-c]", "b", true},
		{"A", "a", false},
	}
	for _, test := range cases {
		pattern, err := New(test.pattern)
		if err != nil {
			t.Fatal(err)
		}
		if got := pattern.Match(test.path); got != test.expected {
			t.Errorf("%q ~ %q: got %v", test.pattern, test.path, got)
		}
	}
}

func TestRejectsInvalidPatterns(t *testing.T) {
	for pattern, message := range map[string]string{
		"[":    "Pattern syntax error near position 0: invalid range pattern",
		"a**b": "Pattern syntax error near position 0: recursive wildcards must form a single path component",
		"***":  "Pattern syntax error near position 2: wildcards are either regular `*` or recursive `**`",
		"**b":  "Pattern syntax error near position 2: recursive wildcards must form a single path component",
	} {
		_, err := New(pattern)
		if err == nil || err.Error() != message {
			t.Errorf("%q: %v", pattern, err)
		}
	}
}
