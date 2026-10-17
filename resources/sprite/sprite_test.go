package sprite

import (
	"encoding/binary"
	"fmt"
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

// riff builds a WebP container from tagged chunks.
func riff(chunks ...string) []byte {
	body := []byte("WEBP")
	for index := 0; index < len(chunks); index += 2 {
		data := chunks[index+1]
		body = append(body, chunks[index]...)
		body = binary.LittleEndian.AppendUint32(body, uint32(len(data)))
		body = append(body, data...)
		if len(data)%2 == 1 {
			body = append(body, 0)
		}
	}
	return append(binary.LittleEndian.AppendUint32([]byte("RIFF"), uint32(len(body))), body...)
}

func TestReadsEmbeddedDescriptor(t *testing.T) {
	data := riff("VP8X", "odd", "extr", "1\n\n192 192\n1\n0 0 122 151 30 17 0 \n")
	sprite, ok, err := Embedded(data)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if !slices.Equal(sprite.Textures, []string{""}) || sprite.Width != 192 ||
		sprite.Frames[0] != (Frame{Width: 122, Height: 151, OffsetX: 30, OffsetY: 17}) {
		t.Fatalf("unexpected sprite %+v", sprite)
	}
	for name, data := range map[string][]byte{
		"png":        []byte("\x89PNG\r\n\x1a\n...."),
		"plain webp": riff("VP8L", "pixels"),
	} {
		if _, ok, err := Embedded(data); ok || err != nil {
			t.Errorf("%s: ok=%v err=%v", name, ok, err)
		}
	}
	truncated := riff("extr", "1\n\n1 1\n1\n0 0 1 1 0 0 0\n")
	if _, _, err := Embedded(truncated[:len(truncated)-4]); blitz.AsError(err).Code != "SPRITE_INVALID" {
		t.Fatalf("expected SPRITE_INVALID, got %v", err)
	}
	if _, _, err := Embedded(riff("extr", "1\n\n1 1\n1\n0 0 2 2 0 0 0\n")); blitz.AsError(err).Code != "SPRITE_INVALID" {
		t.Fatalf("expected SPRITE_INVALID, got %v", err)
	}
}

func TestRejectsOversizedCanvasAndOverflowingFrame(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	for _, descriptor := range []string{
		fmt.Sprintf("1\nt.png\n%d 2\n1\n0 0 1 1 0 0 0\n", maxInt),
		"1\nt.png\n8192 8192\n1\n0 0 1 1 0 0 0\n",
		fmt.Sprintf("1\nt.png\n10 10\n1\n0 0 2 1 %d 0 0\n", maxInt),
		fmt.Sprintf("1\nt.png\n10 10\n1\n0 0 %d 1 2 0 0\n", maxInt),
	} {
		if _, err := Parse([]byte(descriptor)); err == nil || blitz.AsError(err).Code != "SPRITE_INVALID" {
			t.Fatalf("expected SPRITE_INVALID, got %v", err)
		}
	}
}
