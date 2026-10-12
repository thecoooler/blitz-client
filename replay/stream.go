package replay

import (
	"encoding/binary"
	"fmt"
	"math"
	"unicode/utf8"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/internal/jsonx"
	"github.com/thecoooler/blitz-client/internal/rs"
)

// MaxPayloadBytes limits a single packet payload.
const MaxPayloadBytes = 16 * 1024 * 1024

const streamMagic = 0x12345678

// StreamEventKind identifies a stream event.
type StreamEventKind int

const (
	EventHeader StreamEventKind = iota
	EventPacket
	EventReset
)

// StreamEvent is a header, packet, or reset event.
type StreamEvent struct {
	Kind StreamEventKind
	// Header fields.
	ClientVersion string
	HeaderBytes   uint64
	// Packet fields.
	Seq        int
	Offset     uint64
	PacketType uint32
	Packet     *Packet
	// Reset reason.
	Reason string
}

func (e *StreamEvent) fields() orderedObject {
	switch e.Kind {
	case EventHeader:
		return orderedObject{{"event", "header"}, {"client_version", e.ClientVersion}, {"header_bytes", e.HeaderBytes}}
	case EventPacket:
		return append(orderedObject{{"event", "packet"}, {"seq", e.Seq}, {"offset", e.Offset}, {"packet_type", e.PacketType}},
			e.Packet.fields(jsonx.F32(e.Packet.ClockSecs))...)
	}
	return orderedObject{{"event", "reset"}, {"reason", e.Reason}}
}

// MarshalJSON encodes the internally tagged event.
func (e *StreamEvent) MarshalJSON() ([]byte, error) { return e.fields().MarshalJSON() }

func dataInvalid(message string) error { return blitz.NewError("REPLAY_DATA_INVALID", message) }

// StreamDecoder frames data.replay bytes incrementally, buffering only an
// incomplete frame. Call Finish only for completed input, never at a
// temporary EOF of a growing file. Callback errors terminate decoding.
type StreamDecoder struct {
	pending       []byte
	clientVersion *string
	offset        uint64
	seq           int
}

// ClientVersion returns the header's client version, or "" before the header.
func (d *StreamDecoder) ClientVersion() string {
	if d.clientVersion == nil {
		return ""
	}
	return *d.clientVersion
}

// Feed decodes bytes and emits complete frames.
func (d *StreamDecoder) Feed(data []byte, emit func(StreamEvent) error) error {
	// Chunk large inputs so an entire replay is not copied into pending.
	for len(data) > 0 {
		chunk := data[:min(len(data), 64*1024)]
		data = data[len(chunk):]
		d.pending = append(d.pending, chunk...)
		consumed := 0
		err := d.drain(&consumed, emit)
		d.pending = append(d.pending[:0], d.pending[consumed:]...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *StreamDecoder) drain(consumed *int, emit func(StreamEvent) error) error {
	if d.clientVersion == nil {
		if len(d.pending) >= 4 && binary.LittleEndian.Uint32(d.pending) != streamMagic {
			return dataInvalid("invalid data.replay magic")
		}
		if len(d.pending) < 13 {
			return nil
		}
		lengthOffset := 13 + int(d.pending[12])
		if lengthOffset >= len(d.pending) {
			return nil
		}
		versionStart := lengthOffset + 1
		headerLength := versionStart + int(d.pending[lengthOffset]) + 1
		if len(d.pending) < headerLength {
			return nil
		}
		raw := d.pending[versionStart : headerLength-1]
		if !utf8.Valid(raw) {
			return dataInvalid("invalid client version: " + rs.UTF8Error(raw))
		}
		version := string(raw)
		d.clientVersion = &version
		*consumed = headerLength
		d.offset = uint64(headerLength)
		if err := emit(StreamEvent{Kind: EventHeader, ClientVersion: version, HeaderBytes: d.offset}); err != nil {
			return err
		}
	}
	for len(d.pending)-*consumed >= 4 {
		raw := d.pending[*consumed:]
		length := int(binary.LittleEndian.Uint32(raw))
		if length > MaxPayloadBytes {
			return dataInvalid(fmt.Sprintf("payload length %d exceeds %d at offset %d", length, MaxPayloadBytes, d.offset))
		}
		frameLength := 12 + length
		if len(raw) < frameLength {
			break
		}
		packetType := binary.LittleEndian.Uint32(raw[4:])
		clock := math.Float32frombits(binary.LittleEndian.Uint32(raw[8:]))
		if math.IsNaN(float64(clock)) || math.IsInf(float64(clock), 0) {
			return dataInvalid(fmt.Sprintf("non-finite packet clock at offset %d", d.offset))
		}
		rawPayload := append([]byte(nil), raw[12:frameLength]...)
		payload, err := NewPayload(packetType, rawPayload)
		if err != nil {
			return dataInvalid(fmt.Sprintf("cannot decode packet %d at offset %d: %v", d.seq, d.offset, err))
		}
		event := StreamEvent{
			Kind: EventPacket, Seq: d.seq, Offset: d.offset, PacketType: packetType,
			Packet: &Packet{ClockSecs: clock, Payload: payload, RawPayload: rawPayload},
		}
		*consumed += frameLength
		d.offset += uint64(frameLength)
		d.seq++
		if err := emit(event); err != nil {
			return err
		}
	}
	return nil
}

// Finish rejects a missing header or a buffered incomplete frame.
func (d *StreamDecoder) Finish() error {
	if d.clientVersion == nil || len(d.pending) > 0 {
		return dataInvalid(fmt.Sprintf("truncated data.replay at offset %d (%d buffered bytes)", d.offset, len(d.pending)))
	}
	return nil
}
