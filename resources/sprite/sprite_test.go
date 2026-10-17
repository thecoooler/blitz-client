package sprite

import (
	"slices"
	"testing"

	blitz "github.com/thecoooler/blitz-client"
)

func TestParsesDescriptor(t *testing.T) {
	sprite, err := Parse([]byte("1\r\ntexture0@2x.packed.webp\n360 360\n1\n973 308 303 304 36 31 0 frame0\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(sprite.Textures, []string{"texture0@2x.packed.webp"}) || sprite.Width != 360 || sprite.Height != 360 {
		t.Fatalf("unexpected sprite %+v", sprite)
	}
	want := Frame{X: 973, Y: 308, Width: 303, Height: 304, OffsetX: 36, OffsetY: 31, Texture: 0, Name: "frame0"}
	if len(sprite.Frames) != 1 || sprite.Frames[0] != want {
		t.Fatalf("unexpected frames %+v", sprite.Frames)
	}
}

func TestRejectsInvalidDescriptors(t *testing.T) {
	for name, source := range map[string]string{
		"empty":           "",
		"no textures":     "0\n10 10\n1\n0 0 1 1 0 0 0\n",
		"missing frames":  "1\nt.png\n10 10\n2\n0 0 1 1 0 0 0\n",
		"short frame":     "1\nt.png\n10 10\n1\n0 0 1 1 0 0\n",
		"text number":     "1\nt.png\n10 ten\n1\n0 0 1 1 0 0 0\n",
		"outside canvas":  "1\nt.png\n10 10\n1\n0 0 8 8 4 0 0\n",
		"missing texture": "1\nt.png\n10 10\n1\n0 0 1 1 0 0 1\n",
		"empty frame":     "1\nt.png\n10 10\n1\n0 0 0 1 0 0 0\n",
	} {
		if _, err := Parse([]byte(source)); blitz.AsError(err) == nil || blitz.AsError(err).Code != "SPRITE_INVALID" {
			t.Errorf("%s: expected SPRITE_INVALID, got %v", name, err)
		}
	}
}
