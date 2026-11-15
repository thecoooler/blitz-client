package replay

import (
	"encoding/binary"
	"sort"
)

// A vehicle transform packet reports one vehicle's world placement. The server
// streams it only while the author's client keeps that vehicle, so a position
// stops updating when the vehicle leaves view: the retained report is its last
// known location, not a live one.
//
// Verified layouts are the unknown packet type 10 with a 49-byte payload: the
// vehicle entity ID as a u32 prefix, then the position vector at offset 12.
// The same layout was checked against 26.8.0_ruby_apple telemetry, the bundled
// 26.10.0_ruby_apple recording and current 26.10.1_ruby_apple recordings.
const (
	vehicleTransformType = 10
	vehicleTransformSize = 49
)

// SupportsVehiclePositions reports whether the vehicle transform layout is
// verified for the client version. Other versions keep their packets raw, like
// every other unverified telemetry layout.
func SupportsVehiclePositions(version string) bool {
	switch version {
	case "26.10.0_ruby_apple", "26.10.1_ruby_apple", "26.8.0_ruby_apple":
		return true
	}
	return false
}

// vehiclePositionDecoder retains the last placement reported for each vehicle
// entity. Only the last report per entity is kept, so a battle's whole history
// never accumulates.
type vehiclePositionDecoder struct {
	positions map[uint32]LiveVehiclePosition
}

// onPacket records a transform packet that matches the verified layout.
func (d *vehiclePositionDecoder) onPacket(packet *Packet, version string) {
	if !SupportsVehiclePositions(version) || packet == nil {
		return
	}
	raw := packet.RawPayload
	if packet.Payload.Kind != PayloadUnknown || packet.Payload.PacketType != vehicleTransformType || len(raw) != vehicleTransformSize {
		return
	}
	x, okX := readF32(raw, 12)
	y, okY := readF32(raw, 16)
	z, okZ := readF32(raw, 20)
	if !okX || !okY || !okZ {
		return
	}
	if d.positions == nil {
		d.positions = make(map[uint32]LiveVehiclePosition)
	}
	d.positions[binary.LittleEndian.Uint32(raw)] = LiveVehiclePosition{
		EntityID: binary.LittleEndian.Uint32(raw), X: x, Y: y, Z: z, Clock: packet.ClockSecs,
	}
}

// snapshot returns the retained positions of the roster's vehicles, sorted by
// entity ID so repeated polls of the same state compare equal. Entities the
// roster does not know, such as the author's separate player entity, are left
// out: no team could be shown for them.
func (d *vehiclePositionDecoder) snapshot(roster *BattleRoster) []LiveVehiclePosition {
	if len(d.positions) == 0 {
		return nil
	}
	seen := make(map[uint32]bool, len(d.positions))
	out := make([]LiveVehiclePosition, 0, len(d.positions))
	add := func(player ArenaPlayer) {
		if player.VehicleEntityID == nil || seen[*player.VehicleEntityID] {
			return
		}
		seen[*player.VehicleEntityID] = true
		if position, ok := d.positions[*player.VehicleEntityID]; ok {
			out = append(out, position)
		}
	}
	for _, player := range roster.Allies {
		add(player)
	}
	for _, player := range roster.Enemies {
		add(player)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].EntityID < out[j].EntityID })
	return out
}
