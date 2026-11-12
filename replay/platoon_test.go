package replay

import (
	"encoding/binary"
	"testing"
)

func TestPlatoonMembersSelectsAuthorPlatoon(t *testing.T) {
	one, two, zero := uint32(1), uint32(2), uint32(0)
	roster := &BattleRoster{AuthorNickname: "Author", AuthorAccountID: 10, Allies: []ArenaPlayer{
		{AccountID: 10, Nickname: "Author", PlatoonNumber: &one},
		{AccountID: 11, Nickname: "Mate", PlatoonNumber: &one},
		{AccountID: 12, Nickname: "Other", PlatoonNumber: &two},
		{AccountID: 13, Nickname: "Solo", PlatoonNumber: &zero},
	}}
	members := roster.PlatoonMembers()
	if len(members) != 1 || members[0].AccountID != 11 {
		t.Fatalf("same-platoon ally: %+v", members)
	}
	// Nickname identifies the author when the roster carries no account ID.
	roster.AuthorAccountID = 0
	if members := roster.PlatoonMembers(); len(members) != 1 || members[0].AccountID != 11 {
		t.Fatalf("nickname fallback: %+v", members)
	}
	roster.Allies[0].PlatoonNumber = nil
	if members := roster.PlatoonMembers(); members != nil {
		t.Fatalf("author without a platoon: %+v", members)
	}
	var absent *BattleRoster
	if absent.PlatoonMembers() != nil {
		t.Fatal("missing roster returned members")
	}
	if members := (&BattleRoster{AuthorNickname: "Nope", AuthorAccountID: 99}).PlatoonMembers(); members != nil {
		t.Fatalf("missing author: %+v", members)
	}
}

func TestLivePlatoonDamageAndResults(t *testing.T) {
	own, mate, other, enemy := uint32(11), uint32(12), uint32(13), uint32(22)
	platoon, otherPlatoon := uint32(4), uint32(5)
	p := livePlayers{author: "Author", version: "26.10.1_ruby", players: map[uint32]ArenaPlayer{
		1: {AccountID: 1, Nickname: "Author", TeamNumber: 1, VehicleEntityID: &own, PlatoonNumber: &platoon},
		2: {AccountID: 2, Nickname: "Mate", TeamNumber: 1, VehicleEntityID: &mate, PlatoonNumber: &platoon},
		3: {AccountID: 3, Nickname: "Other", TeamNumber: 1, VehicleEntityID: &other, PlatoonNumber: &otherPlatoon},
		9: {AccountID: 9, Nickname: "Enemy", TeamNumber: 2, VehicleEntityID: &enemy},
	}}
	feed := func(source, target uint32, amount uint16) {
		t.Helper()
		raw := make([]byte, 33)
		binary.LittleEndian.PutUint32(raw, target)
		binary.LittleEndian.PutUint32(raw[4:], 9)
		binary.LittleEndian.PutUint32(raw[8:], 21)
		binary.LittleEndian.PutUint32(raw[12:], source)
		binary.LittleEndian.PutUint32(raw[16:], target)
		binary.LittleEndian.PutUint16(raw[30:], amount)
		raw[32] = 1
		packet := &Packet{Payload: Payload{Kind: PayloadEntityMethod, EntityMethod: &EntityMethod{SubType: 9}}, RawPayload: raw}
		if err := p.event(FollowEvent{Event: StreamEvent{Kind: EventPacket, Packet: packet}}); err != nil {
			t.Fatal(err)
		}
	}
	feed(own, enemy, 500)
	feed(mate, enemy, 600)
	feed(mate, own, 300) // Team damage is not attributed as dealt damage.
	feed(other, enemy, 100)

	roster, damage := p.snapshot("")
	members := roster.PlatoonMembers()
	if len(members) != 1 || members[0].Nickname != "Mate" {
		t.Fatalf("platoon members: %+v", members)
	}
	if damage.Dealt == nil || *damage.Dealt != 500 {
		t.Fatalf("author damage: %+v", damage)
	}
	for _, ally := range roster.Allies {
		dealt := ally.Damage.Dealt
		want := map[string]uint64{"Author": 500, "Mate": 600, "Other": 100}[ally.Nickname]
		if dealt == nil || *dealt != want {
			t.Fatalf("%s live damage: %+v", ally.Nickname, ally.Damage)
		}
		if ally.Nickname != "Author" && (ally.Damage.Assisted != nil || ally.Damage.Blocked != nil) {
			t.Fatalf("%s fabricated author-only counters: %+v", ally.Nickname, ally.Damage)
		}
	}
	if *members[0].Damage.Dealt != 600 || members[0].Damage.Assisted != nil || members[0].Damage.Blocked != nil {
		t.Fatalf("platoon mate before results: %+v", members[0].Damage)
	}

	p.results = &BattleResults{Author: Author{AccountID: 1, DamageDealt: 520}, PlayerResults: []PlayerResults{
		{Info: PlayerResultsInfo{AccountID: 2, DamageDealt: 620, DamageAssisted1: 430, DamageBlocked: 270}},
	}}
	roster, _ = p.snapshot("")
	members = roster.PlatoonMembers()
	if len(members) != 1 || *members[0].Damage.Dealt != 620 ||
		*members[0].Damage.Assisted != 430 || *members[0].Damage.Blocked != 270 {
		t.Fatalf("platoon mate after results: %+v", members)
	}
	// Another author's results must not replace live values.
	p.results = &BattleResults{Author: Author{AccountID: 99, DamageDealt: 9999}}
	roster, _ = p.snapshot("")
	members = roster.PlatoonMembers()
	if len(members) != 1 || *members[0].Damage.Dealt != 600 || members[0].Damage.Assisted != nil {
		t.Fatalf("foreign results: %+v", members)
	}
}
