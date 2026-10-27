// Package replay analyzes complete replays and decodes live replay streams
// incrementally.
package replay

import (
	"encoding/binary"
	"fmt"
	"unicode/utf8"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/internal/jsonx"
	"github.com/thecoooler/blitz-client/internal/pickle"
	"github.com/thecoooler/blitz-client/internal/rs"
)

// DefaultReplayDir re-exports blitzclient.DefaultReplayDir.
func DefaultReplayDir() (string, error) { return blitz.DefaultReplayDir() }

// Packet is a decoded frame that retains every original payload byte.
type Packet struct {
	ClockSecs  float32
	Payload    Payload
	RawPayload []byte
}

// PayloadKind identifies a Payload variant.
type PayloadKind int

const (
	PayloadUnknown PayloadKind = iota
	PayloadBasePlayerCreate
	PayloadEntityMethod
)

// Payload is a decoded packet payload.
type Payload struct {
	Kind PayloadKind
	// PacketType is set for unknown payloads.
	PacketType uint32
	// BasePlayerCreate fields.
	AuthorNickname string
	ArenaUniqueID  uint64
	ArenaTypeID    uint32
	Arguments      *BasePlayerCreate
	// EntityMethod is set for EntityMethod payloads.
	EntityMethod *EntityMethod
}

// EntityMethod is a decoded entity method call.
type EntityMethod struct {
	// UpdateArena is nil for unknown methods, which keep their SubType.
	UpdateArena *UpdateArena
	FieldNumber uint64
	SubType     uint32
}

func payloadInvalid(message string) error { return blitz.NewError("REPLAY_PAYLOAD_INVALID", message) }

// NewPayload decodes a payload of the given packet type.
func NewPayload(packetType uint32, raw []byte) (Payload, error) {
	return NewPayloadForVersion(packetType, raw, "")
}

// NewPayloadForVersion selects method IDs for the stream header.
func NewPayloadForVersion(packetType uint32, raw []byte, version string) (Payload, error) {
	reader := &payloadReader{remaining: raw}
	switch packetType {
	case 0:
		if _, err := reader.take(10); err != nil {
			return Payload{}, err
		}
		nickname, err := reader.string()
		if err != nil {
			return Payload{}, err
		}
		arenaUniqueID, err := reader.u64()
		if err != nil {
			return Payload{}, err
		}
		arenaTypeID, err := reader.u32()
		if err != nil {
			return Payload{}, err
		}
		length, err := reader.quirkyLength()
		if err != nil {
			return Payload{}, err
		}
		data, err := reader.take(length)
		if err != nil {
			return Payload{}, err
		}
		value, err := pickle.Decode(data)
		var arguments *BasePlayerCreate
		if err == nil {
			arguments, err = decodeBasePlayerCreate(value)
		}
		if err != nil {
			return Payload{}, payloadInvalid(fmt.Sprintf("cannot decode arena context pickle: %v", err))
		}
		return Payload{Kind: PayloadBasePlayerCreate, AuthorNickname: nickname, ArenaUniqueID: arenaUniqueID,
			ArenaTypeID: arenaTypeID, Arguments: arguments}, nil
	case 8:
		method, err := NewEntityMethodForVersion(raw, version)
		if err != nil {
			return Payload{}, err
		}
		return Payload{Kind: PayloadEntityMethod, EntityMethod: method}, nil
	}
	return Payload{Kind: PayloadUnknown, PacketType: packetType}, nil
}

// NewEntityMethod decodes an EntityMethod payload.
func NewEntityMethod(raw []byte) (*EntityMethod, error) { return NewEntityMethodForVersion(raw, "") }

func NewEntityMethodForVersion(raw []byte, version string) (*EntityMethod, error) {
	reader := &payloadReader{remaining: raw}
	if _, err := reader.u32(); err != nil { // Receiver entity ID.
		return nil, err
	}
	subType, err := reader.u32()
	if err != nil {
		return nil, err
	}
	updateArenaMethod := uint32(47)
	switch version {
	case "26.10.0_ruby_apple", "26.10.1_ruby":
		updateArenaMethod = 55
	}
	if subType != updateArenaMethod {
		return &EntityMethod{SubType: subType}, nil
	}
	if _, err := reader.u32(); err != nil { // Method envelope length.
		return nil, err
	}
	fieldNumber, err := reader.varint()
	if err != nil {
		return nil, err
	}
	length, err := reader.quirkyLength()
	if err != nil {
		return nil, err
	}
	data, err := reader.take(length)
	if err != nil {
		return nil, err
	}
	arena, err := decodeUpdateArena(data)
	if err != nil {
		return nil, payloadInvalid(fmt.Sprintf("cannot decode UpdateArena protobuf: %v", err))
	}
	return &EntityMethod{UpdateArena: arena, FieldNumber: fieldNumber, SubType: subType}, nil
}

// payloadReader reads borrowed slices so length fields cannot cause unbounded
// allocations.
type payloadReader struct {
	remaining []byte
}

func (r *payloadReader) take(length int) ([]byte, error) {
	if length > len(r.remaining) {
		return nil, payloadInvalid(fmt.Sprintf("truncated payload: need %d bytes, have %d", length, len(r.remaining)))
	}
	out := r.remaining[:length]
	r.remaining = r.remaining[length:]
	return out, nil
}

func (r *payloadReader) u8() (byte, error) {
	b, err := r.take(1)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

func (r *payloadReader) u16() (uint16, error) {
	b, err := r.take(2)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(b), nil
}

func (r *payloadReader) u32() (uint32, error) {
	b, err := r.take(4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(b), nil
}

func (r *payloadReader) u64() (uint64, error) {
	b, err := r.take(8)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(b), nil
}

func (r *payloadReader) string() (string, error) {
	length, err := r.u8()
	if err != nil {
		return "", err
	}
	data, err := r.take(int(length))
	if err != nil {
		return "", err
	}
	if !utf8.Valid(data) {
		return "", payloadInvalid("invalid payload string: " + rs.UTF8Error(data))
	}
	return string(data), nil
}

func (r *payloadReader) quirkyLength() (int, error) {
	first, err := r.u8()
	if err != nil {
		return 0, err
	}
	if first != 0xff {
		return int(first), nil
	}
	length, err := r.u16()
	if err != nil {
		return 0, err
	}
	terminator, err := r.u8()
	if err != nil {
		return 0, err
	}
	if terminator != 0 {
		return 0, payloadInvalid("invalid extended-length terminator")
	}
	return int(length), nil
}

func (r *payloadReader) varint() (uint64, error) {
	var result uint64
	for shift := 0; shift < 70; shift += 7 {
		b, err := r.u8()
		if err != nil {
			return 0, err
		}
		if shift == 63 && b > 1 {
			return 0, payloadInvalid("protobuf varint overflow")
		}
		result |= uint64(b&0x7f) << shift
		if b&0x80 == 0 {
			return result, nil
		}
	}
	return 0, payloadInvalid("protobuf varint overflow")
}

func hexString(data []byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, len(data)*2)
	for index, b := range data {
		out[index*2] = digits[b>>4]
		out[index*2+1] = digits[b&15]
	}
	return string(out)
}

// orderedObject is a JSON object whose keys keep insertion order, matching
// serde's derived struct serialization.
type orderedObject []field

type field struct {
	key   string
	value any
}

func (o orderedObject) MarshalJSON() ([]byte, error) {
	out := []byte{'{'}
	for index, entry := range o {
		if index > 0 {
			out = append(out, ',')
		}
		key, err := jsonx.Marshal(entry.key)
		if err != nil {
			return nil, err
		}
		value, err := jsonx.Marshal(entry.value)
		if err != nil {
			return nil, err
		}
		out = append(append(append(out, key...), ':'), value...)
	}
	return append(out, '}'), nil
}

func (o orderedObject) JSONValue() any {
	out := jsonx.Object{}
	for _, entry := range o {
		out[entry.key] = jsonx.ToValue(entry.value)
	}
	return out
}

func (p Payload) fields() orderedObject {
	switch p.Kind {
	case PayloadBasePlayerCreate:
		return orderedObject{{"BasePlayerCreate", orderedObject{
			{"author_nickname", p.AuthorNickname},
			{"arena_unique_id", p.ArenaUniqueID},
			{"arena_type_id", p.ArenaTypeID},
			{"arguments", p.Arguments},
		}}}
	case PayloadEntityMethod:
		return orderedObject{{"EntityMethod", p.EntityMethod.fields()}}
	}
	return orderedObject{{"Unknown", orderedObject{{"packet_type", p.PacketType}}}}
}

func (m *EntityMethod) fields() orderedObject {
	if m.UpdateArena != nil {
		return orderedObject{{"UpdateArena", orderedObject{
			{"field_number", m.FieldNumber},
			{"arguments", m.UpdateArena},
		}}}
	}
	return orderedObject{{"Unknown", orderedObject{{"sub_type", m.SubType}}}}
}

// MarshalJSON encodes the externally tagged enum.
func (p Payload) MarshalJSON() ([]byte, error) { return p.fields().MarshalJSON() }

// JSONValue implements jsonx.Valuer.
func (p Payload) JSONValue() any { return p.fields().JSONValue() }

func (p *Packet) fields(clock any) orderedObject {
	return orderedObject{
		{"clock", clock},
		{"payload", p.Payload},
		{"raw_payload", hexString(p.RawPayload)},
	}
}

// MarshalJSON encodes the packet with its f32 clock.
func (p *Packet) MarshalJSON() ([]byte, error) { return p.fields(jsonx.F32(p.ClockSecs)).MarshalJSON() }

// JSONValue implements jsonx.Valuer, widening the clock like serde_json::Value.
func (p *Packet) JSONValue() any { return p.fields(jsonx.F64(p.ClockSecs)).JSONValue() }
