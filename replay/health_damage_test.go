package replay

import (
	"encoding/binary"
	"testing"
)

func healthSpawn(target uint32, hp uint16) *Packet {
	raw := make([]byte, 53)
	binary.LittleEndian.PutUint32(raw, target)
	binary.LittleEndian.PutUint16(raw[4:], 2)
	binary.LittleEndian.PutUint32(raw[38:], uint32(len(raw)-42))
	raw[42], raw[43], raw[45], raw[47], raw[50] = 4, 0, 1, 2, 3
	binary.LittleEndian.PutUint16(raw[51:], hp)
	return &Packet{Payload: Payload{Kind: PayloadUnknown, PacketType: 5}, RawPayload: raw}
}
func healthProperty(target uint32, hp uint16, clock float32) *Packet {
	raw := make([]byte, 14)
	binary.LittleEndian.PutUint32(raw, target)
	binary.LittleEndian.PutUint32(raw[4:], 3)
	binary.LittleEndian.PutUint32(raw[8:], 2)
	binary.LittleEndian.PutUint16(raw[12:], hp)
	return &Packet{ClockSecs: clock, Payload: Payload{Kind: PayloadUnknown, PacketType: 7}, RawPayload: raw}
}
func healthChange(target, source uint32, hp uint16, reason byte, clock float32) *Packet {
	raw := make([]byte, 19)
	binary.LittleEndian.PutUint32(raw, target)
	binary.LittleEndian.PutUint32(raw[4:], 2)
	binary.LittleEndian.PutUint32(raw[8:], 7)
	binary.LittleEndian.PutUint16(raw[12:], hp)
	binary.LittleEndian.PutUint32(raw[14:], source)
	raw[18] = reason
	return &Packet{ClockSecs: clock, Payload: Payload{Kind: PayloadEntityMethod, EntityMethod: &EntityMethod{SubType: 2}}, RawPayload: raw}
}

func TestHealthDamageCountsRammingAndSameFrameUpdatesOnce(t *testing.T) {
	for _, version := range []string{"26.10.0_ruby_apple", "26.10.1_ruby"} {
		t.Run(version, func(t *testing.T) {
			d := healthDamageDecoder{}
			d.onPacket(healthSpawn(22, 2070), version)
			property := healthProperty(22, 1904, 73)
			d.onPacket(property, version)
			d.onPacket(property, version)
			change := healthChange(22, 11, 1904, 2, 73)
			hit := d.onPacket(change, version)
			if hit == nil || hit.Damage != 166 || hit.SourceEntityID != 11 || hit.TargetEntityID != 22 {
				t.Fatalf("ramming: %+v", hit)
			}
			if d.onPacket(change, version) != nil {
				t.Fatal("duplicate health method counted twice")
			}
			if d.onPacket(property, version) != nil || d.onPacket(change, version) != nil {
				t.Fatal("duplicate property/method pair counted twice")
			}
			// The method can arrive first; the following property must not repeat it.
			hit = d.onPacket(healthChange(22, 11, 1804, 1, 74), version)
			if hit == nil || hit.Damage != 100 {
				t.Fatalf("damage before property: %+v", hit)
			}
			d.onPacket(healthProperty(22, 1804, 74), version)
			if d.onPacket(healthChange(22, 11, 1804, 1, 74), version) != nil {
				t.Fatal("late property repeated damage")
			}
			// Negative destroyed-HP sentinels never add overkill damage.
			hit = d.onPacket(healthChange(22, 11, 65533, 0, 75), version)
			if hit == nil || hit.Damage != 1804 {
				t.Fatalf("destroyed HP: %+v", hit)
			}
		})
	}
}

func TestHealthDamageResyncHealingAndMissingBaseline(t *testing.T) {
	d := healthDamageDecoder{}
	version := counterVersion
	if d.onPacket(healthChange(22, 11, 900, 2, 1), version) != nil {
		t.Fatal("damage invented without previous HP")
	}
	d.onPacket(healthSpawn(22, 500), version) // A visibility resync establishes a fresh baseline.
	hit := d.onPacket(healthChange(22, 11, 400, 2, 2), version)
	if hit == nil || hit.Damage != 100 {
		t.Fatalf("resync: %+v", hit)
	}
	d.onPacket(healthProperty(22, 200, 3), version) // Unattributed loss cannot belong to a later attacker.
	hit = d.onPacket(healthChange(22, 11, 150, 2, 4), version)
	if hit == nil || hit.Damage != 50 {
		t.Fatalf("property resync: %+v", hit)
	}
	if d.onPacket(healthChange(22, 11, 300, 0, 5), version) != nil {
		t.Fatal("healing counted as damage")
	}
	if d.onPacket(healthChange(22, 0, 250, 0, 6), version) != nil {
		t.Fatal("unknown attacker counted")
	}
	if d.onPacket(healthChange(22, 22, 200, 0, 7), version) != nil {
		t.Fatal("self damage counted")
	}
}

func TestHealthDamageRejectsMalformedAndUnverifiedLayouts(t *testing.T) {
	for _, packet := range []*Packet{healthSpawn(22, 1000), healthProperty(22, 500, 1), healthChange(22, 11, 500, 2, 1)} {
		d := healthDamageDecoder{}
		d.onPacket(healthSpawn(22, 1000), counterVersion)
		for n := 0; n < len(packet.RawPayload); n++ {
			truncated := *packet
			truncated.RawPayload = packet.RawPayload[:n]
			if d.onPacket(&truncated, counterVersion) != nil {
				t.Fatalf("accepted truncated packet %d", n)
			}
		}
		if d.onPacket(packet, "unverified") != nil {
			t.Fatal("unsupported layout accepted")
		}
	}
}

func TestLiveDamageCombinesHealthAndHitsWithoutDoubleCounting(t *testing.T) {
	own, enemy, ally := uint32(11), uint32(22), uint32(33)
	p := livePlayers{author: "Test", version: counterVersion, players: map[uint32]ArenaPlayer{
		42: {AccountID: 42, Nickname: "Test", TeamNumber: 1, VehicleEntityID: &own},
		99: {AccountID: 99, Nickname: "Enemy", TeamNumber: 2, VehicleEntityID: &enemy},
		77: {AccountID: 77, Nickname: "Ally", TeamNumber: 1, VehicleEntityID: &ally},
	}, pairs: map[[2]uint32]uint64{{11, 22}: 500}}
	for _, packet := range []*Packet{healthSpawn(enemy, 1000), healthChange(enemy, own, 500, 0, 1), healthChange(enemy, own, 334, 2, 2), healthSpawn(ally, 1000), healthChange(ally, own, 0, 2, 3), healthSpawn(own, 1000), healthChange(own, enemy, 0, 2, 4)} {
		if err := p.event(FollowEvent{Event: StreamEvent{Kind: EventPacket, Packet: packet}}); err != nil {
			t.Fatal(err)
		}
	}
	_, damage := p.snapshot("")
	if damage.Dealt == nil || *damage.Dealt != 666 {
		t.Fatalf("live combined damage: %+v", damage)
	}
	p.event(FollowEvent{Event: StreamEvent{Kind: EventReset}})
	if len(p.health.vehicles) != 0 || len(p.healthPairs) != 0 {
		t.Fatal("reset retained health history")
	}
}

func TestLiveRosterDestroyedStateFollowsHealthAndResets(t *testing.T) {
	for _, version := range []string{"26.10.0_ruby_apple", "26.10.1_ruby"} {
		t.Run(version, func(t *testing.T) {
			own, enemy := uint32(11), uint32(22)
			players := map[uint32]ArenaPlayer{
				42: {AccountID: 42, Nickname: "Test", TeamNumber: 1, VehicleEntityID: &own},
				99: {AccountID: 99, Nickname: "Enemy", TeamNumber: 2, VehicleEntityID: &enemy},
			}
			p := livePlayers{author: "Test", version: version, players: players}
			feed := func(packet *Packet) {
				t.Helper()
				if err := p.event(FollowEvent{Event: StreamEvent{Kind: EventPacket, Packet: packet}}); err != nil {
					t.Fatal(err)
				}
			}
			roster, _ := p.snapshot("")
			if roster.Allies[0].Destroyed || roster.Enemies[0].Destroyed {
				t.Fatal("unknown HP treated as destroyed")
			}
			feed(healthSpawn(enemy, 1000))
			feed(healthChange(enemy, own, 0, 0, 1))
			roster, _ = p.snapshot("")
			if !roster.Enemies[0].Destroyed || roster.Allies[0].Destroyed {
				t.Fatal("enemy destruction not reflected")
			}
			roster.Enemies[0].Destroyed = false
			again, _ := p.snapshot("")
			if !again.Enemies[0].Destroyed || p.players[99].Destroyed {
				t.Fatal("snapshot status leaks into source roster")
			}
			feed(healthProperty(own, 65533, 2))
			roster, _ = p.snapshot("")
			if !roster.Allies[0].Destroyed {
				t.Fatal("negative HP sentinel did not mark author destroyed")
			}
			// Revival or a new live vehicle snapshot moves a player back to live state.
			feed(healthSpawn(enemy, 1000))
			roster, _ = p.snapshot("")
			if roster.Enemies[0].Destroyed {
				t.Fatal("respawn retained destroyed state")
			}
			p.event(FollowEvent{Event: StreamEvent{Kind: EventReset}})
			p.author, p.version, p.players = "Test", version, players
			roster, _ = p.snapshot("")
			if roster.Allies[0].Destroyed || roster.Enemies[0].Destroyed {
				t.Fatal("destroyed state survived replay reset")
			}
		})
	}
}
