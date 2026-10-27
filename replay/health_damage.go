package replay

import "encoding/binary"

// Vehicle health property updates can precede onHealthChanged in the same
// frame. Retain that frame's previous HP, then consume it once with the method
// carrying the attacker. Property-only resyncs establish a new baseline; they
// must never attribute off-screen HP loss to a later attacker.
type vehicleHealth struct {
	hp, before  uint16
	clock       float32
	beforeKnown bool
}

type healthDamageDecoder struct{ vehicles map[uint32]vehicleHealth }

func remainingHealth(raw uint16) uint16 {
	// Destroyed vehicles can report negative sentinel HP, e.g. -3.
	if int16(raw) < 0 {
		return 0
	}
	return raw
}

func (d *healthDamageDecoder) onPacket(packet *Packet, version string) *DamageEvent {
	if !SupportsDamageEvents(version) || packet == nil {
		return nil
	}
	raw := packet.RawPayload
	// Vehicle creation/resync (type 2) starts its property stream with
	// indices 0:u8, 1:u8, 2:u16, 3:i16 (health). Validate the envelope and
	// each index rather than searching a variable-length payload for HP.
	spawn := packet.Payload.Kind == PayloadUnknown && packet.Payload.PacketType == 5 && len(raw) >= 53 &&
		binary.LittleEndian.Uint16(raw[4:]) == 2 && binary.LittleEndian.Uint32(raw[38:]) == uint32(len(raw)-42) &&
		raw[42] >= 4 && raw[43] == 0 && raw[45] == 1 && raw[47] == 2 && raw[50] == 3
	property := packet.Payload.Kind == PayloadUnknown && packet.Payload.PacketType == 7 && len(raw) == 14 &&
		binary.LittleEndian.Uint32(raw[4:]) == 3 && binary.LittleEndian.Uint32(raw[8:]) == 2
	method := packet.Payload.Kind == PayloadEntityMethod && packet.Payload.EntityMethod != nil &&
		packet.Payload.EntityMethod.SubType == 2 && len(raw) == 19 && binary.LittleEndian.Uint32(raw[4:]) == 2 && binary.LittleEndian.Uint32(raw[8:]) == 7
	if !spawn && !property && !method {
		return nil
	}
	if d.vehicles == nil {
		d.vehicles = make(map[uint32]vehicleHealth)
	}
	target := binary.LittleEndian.Uint32(raw)
	if spawn {
		d.vehicles[target] = vehicleHealth{hp: remainingHealth(binary.LittleEndian.Uint16(raw[51:])), clock: packet.ClockSecs}
		return nil
	}
	hp := remainingHealth(binary.LittleEndian.Uint16(raw[12:]))
	previous, known := d.vehicles[target]
	if property {
		if known && previous.clock == packet.ClockSecs && previous.hp == hp {
			return nil
		}
		d.vehicles[target] = vehicleHealth{hp: hp, before: previous.hp, clock: packet.ClockSecs, beforeKnown: known}
		return nil
	}
	before := previous.hp
	if previous.beforeKnown && previous.clock == packet.ClockSecs && previous.hp == hp {
		before = previous.before
	}
	d.vehicles[target] = vehicleHealth{hp: hp, clock: packet.ClockSecs}
	if !known || hp >= before {
		return nil
	}
	source := binary.LittleEndian.Uint32(raw[14:])
	if source == 0 || source == target {
		return nil
	}
	return &DamageEvent{SourceEntityID: source, TargetEntityID: target, Damage: before - hp}
}
