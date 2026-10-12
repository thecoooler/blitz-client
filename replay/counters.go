package replay

import (
	"encoding/binary"
	"math"
	"slices"

	"github.com/thecoooler/blitz-client/internal/jsonx"
)

// BattleCounterClientVersions lists the client versions whose author-only
// battle counters (EntityMethod 14) have been verified.
var BattleCounterClientVersions = []string{"26.8.0_ruby_apple", "26.10.0_ruby_apple"}

// BattleCounters holds the author's cumulative server-reported counters.
type BattleCounters struct {
	AuthorNickname *string `json:"author_nickname"`
	PlayerEntityID *uint32 `json:"player_entity_id"`
	// DamageBlocked is nil until a counter update is received; absence does not imply zero.
	DamageBlocked *uint32 `json:"damage_blocked"`
	// DamageAssisted combines assistance from spotting or immobilizing opponents.
	DamageAssisted *uint32 `json:"damage_assisted"`
}

// BattleCounterKind names a counter.
type BattleCounterKind string

const (
	CounterDamageBlocked  BattleCounterKind = "damage_blocked"
	CounterDamageAssisted BattleCounterKind = "damage_assisted"
)

// BattleCounterUpdate is one counter report.
type BattleCounterUpdate struct {
	PacketIndex int               `json:"packet_index"`
	Clock       jsonx.F32         `json:"clock"`
	EntityID    uint32            `json:"entity_id"`
	RibbonID    uint8             `json:"ribbon_id"`
	Counter     BattleCounterKind `json:"counter"`
	// Value is the absolute value reported by the server, never a sum of hit estimates.
	Value         uint32  `json:"value"`
	PreviousValue *uint32 `json:"previous_value"`
	// Delta is nil on the first update and may be negative for a correction.
	Delta *int64 `json:"delta"`
}

// BattleCounterDecoder tracks counters across packets in stream order.
type BattleCounterDecoder struct {
	counters BattleCounters
}

// SupportsBattleCounters reports whether counters are verified for a version.
func SupportsBattleCounters(version string) bool {
	return slices.Contains(BattleCounterClientVersions, version)
}

// Counters returns the current totals.
func (d *BattleCounterDecoder) Counters() BattleCounters { return d.counters }

// Reset starts a new session.
func (d *BattleCounterDecoder) Reset() { d.counters = BattleCounters{} }

func readU32(data []byte, offset int) (uint32, bool) {
	if offset < 0 || offset+4 > len(data) {
		return 0, false
	}
	return binary.LittleEndian.Uint32(data[offset:]), true
}

// OnPacket feeds a complete packet. The BasePlayerCreate prefix identifies the
// author entity, so vehicle methods cannot affect totals.
func (d *BattleCounterDecoder) OnPacket(packet *Packet, packetIndex int, version string) *BattleCounterUpdate {
	if !SupportsBattleCounters(version) {
		d.Reset()
		return nil
	}
	if packet.Payload.Kind == PayloadBasePlayerCreate {
		nickname := packet.Payload.AuthorNickname
		d.counters = BattleCounters{AuthorNickname: &nickname}
		if id, ok := readU32(packet.RawPayload, 0); ok {
			d.counters.PlayerEntityID = &id
		}
		return nil
	}
	if packet.Payload.Kind != PayloadEntityMethod {
		return nil
	}
	raw := packet.RawPayload
	clock := float64(packet.ClockSecs)
	if len(raw) != 18 || math.IsNaN(clock) || math.IsInf(clock, 0) {
		return nil
	}
	if method, _ := readU32(raw, 4); method != 14 {
		return nil
	}
	if length, _ := readU32(raw, 8); length != 6 {
		return nil
	}
	entityID, _ := readU32(raw, 0)
	if d.counters.PlayerEntityID == nil || *d.counters.PlayerEntityID != entityID {
		return nil
	}
	var counter BattleCounterKind
	var total **uint32
	switch raw[12] {
	case 5:
		counter, total = CounterDamageBlocked, &d.counters.DamageBlocked
	case 17:
		counter, total = CounterDamageAssisted, &d.counters.DamageAssisted
	default:
		return nil
	}
	// Body: ribbon ID at +0, three opaque bytes, cumulative u16 at +4. IDs
	// confirmed in XML/item_defs/ribbons.xml; values checked against
	// battle_results.dat in real 26.8 and 26.10 recordings.
	value := uint32(binary.LittleEndian.Uint16(raw[16:]))
	previous := *total
	stored := value
	*total = &stored
	update := &BattleCounterUpdate{
		PacketIndex: packetIndex, Clock: jsonx.F32(packet.ClockSecs), EntityID: entityID,
		RibbonID: raw[12], Counter: counter, Value: value, PreviousValue: previous,
	}
	if previous != nil {
		delta := int64(value) - int64(*previous)
		update.Delta = &delta
	}
	return update
}
