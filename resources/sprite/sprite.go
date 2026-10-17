// Package sprite parses DAVA sprite descriptors: the .txt resources that
// place named frames from one or more texture atlases onto a sprite canvas.
package sprite

import (
	"fmt"
	"strconv"
	"strings"

	blitz "github.com/thecoooler/blitz-client"
)

// Sprite is a parsed descriptor.
type Sprite struct {
	// Textures are atlas file names relative to the descriptor's directory.
	Textures []string
	Width    int
	Height   int
	Frames   []Frame
}

// Frame is a rectangle of an atlas drawn at an offset on the sprite canvas.
type Frame struct {
	X, Y, Width, Height int
	OffsetX, OffsetY    int
	Texture             int
	Name                string
}

// Parse reads a descriptor:
//
//	<texture count>
//	<texture name>...
//	<width> <height>
//	<frame count>
//	<x> <y> <width> <height> <offset x> <offset y> <texture index> [name]...
//
// Frames must lie inside the canvas and reference a listed texture.
func Parse(data []byte) (*Sprite, error) {
	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	next := 0
	line := func(what string) (string, error) {
		if next >= len(lines) {
			return "", invalid("missing %s", what)
		}
		next++
		return lines[next-1], nil
	}
	numbers := func(what string, count int) ([]int, []string, error) {
		text, err := line(what)
		if err != nil {
			return nil, nil, err
		}
		fields := strings.Fields(text)
		if len(fields) < count {
			return nil, nil, invalid("%s needs %d numbers: %q", what, count, text)
		}
		values := make([]int, count)
		for index := range values {
			if values[index], err = strconv.Atoi(fields[index]); err != nil {
				return nil, nil, invalid("%s has a non-integer value: %q", what, text)
			}
		}
		return values, fields[count:], nil
	}

	textureCount, _, err := numbers("texture count", 1)
	if err != nil {
		return nil, err
	}
	if textureCount[0] < 1 {
		return nil, invalid("texture count must be positive, got %d", textureCount[0])
	}
	sprite := &Sprite{}
	for range textureCount[0] {
		name, err := line("texture name")
		if err != nil {
			return nil, err
		}
		sprite.Textures = append(sprite.Textures, name)
	}
	size, _, err := numbers("sprite size", 2)
	if err != nil {
		return nil, err
	}
	sprite.Width, sprite.Height = size[0], size[1]
	if sprite.Width <= 0 || sprite.Height <= 0 {
		return nil, invalid("sprite size must be positive, got %dx%d", sprite.Width, sprite.Height)
	}
	frameCount, _, err := numbers("frame count", 1)
	if err != nil {
		return nil, err
	}
	if frameCount[0] < 1 {
		return nil, invalid("frame count must be positive, got %d", frameCount[0])
	}
	for index := range frameCount[0] {
		values, rest, err := numbers(fmt.Sprintf("frame %d", index), 7)
		if err != nil {
			return nil, err
		}
		frame := Frame{X: values[0], Y: values[1], Width: values[2], Height: values[3],
			OffsetX: values[4], OffsetY: values[5], Texture: values[6], Name: strings.Join(rest, " ")}
		switch {
		case frame.X < 0 || frame.Y < 0 || frame.Width <= 0 || frame.Height <= 0:
			return nil, invalid("frame %d has an invalid atlas rectangle", index)
		case frame.OffsetX < 0 || frame.OffsetY < 0 ||
			frame.OffsetX+frame.Width > sprite.Width || frame.OffsetY+frame.Height > sprite.Height:
			return nil, invalid("frame %d does not fit the %dx%d sprite", index, sprite.Width, sprite.Height)
		case frame.Texture < 0 || frame.Texture >= len(sprite.Textures):
			return nil, invalid("frame %d references missing texture %d", index, frame.Texture)
		}
		sprite.Frames = append(sprite.Frames, frame)
	}
	return sprite, nil
}

func invalid(format string, args ...any) error {
	return blitz.NewError("SPRITE_INVALID", "invalid sprite descriptor: "+fmt.Sprintf(format, args...))
}
