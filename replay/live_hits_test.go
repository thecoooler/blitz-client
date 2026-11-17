package replay

import "testing"

const hitsVersion = "26.10.0_ruby_apple"

// hitPacket builds one EntityMethod 9 packet carrying a hit message.
func hitPacket(clock float32, source, target uint32, damage uint16, flag byte, result byte) FollowEvent {
	raw := method9BodyBytes(source, target, damage, flag)
	raw[21] = result
	return FollowEvent{Event: StreamEvent{Kind: EventPacket, Packet: &Packet{
		ClockSecs:  clock,
		Payload:    Payload{Kind: PayloadEntityMethod, EntityMethod: &EntityMethod{SubType: 9}},
		RawPayload: raw,
	}}}
}

func headerEvent() FollowEvent {
	return FollowEvent{Event: StreamEvent{Kind: EventHeader, ClientVersion: hitsVersion}}
}

// TestLiveHitsRecordEveryOutcome keeps a hit that applied nothing, because the
// overlay shows bounces as events of their own.
func TestLiveHitsRecordEveryOutcome(t *testing.T) {
	var players livePlayers
	if err := players.event(headerEvent()); err != nil {
		t.Fatal(err)
	}
	for _, event := range []FollowEvent{
		hitPacket(10, 1, 2, 300, 1, 3),
		hitPacket(11, 1, 2, 0, 0, 0),
		hitPacket(12, 2, 1, 400, 0, 1),
		hitPacket(13, 2, 1, 500, 3, 3),
	} {
		if err := players.event(event); err != nil {
			t.Fatal(err)
		}
	}
	hits := players.snapshotHits()
	if len(hits) != 4 {
		t.Fatalf("recorded %d hits, want 4: %+v", len(hits), hits)
	}
	want := []LiveHit{
		{Clock: 10, SourceEntityID: 1, TargetEntityID: 2, Damage: 300, Penetrated: true, ResultCode: 3},
		{Clock: 11, SourceEntityID: 1, TargetEntityID: 2, Damage: 0, Penetrated: false, ResultCode: 0},
		{Clock: 12, SourceEntityID: 2, TargetEntityID: 1, Damage: 400, Penetrated: false, ResultCode: 1},
		{Clock: 13, SourceEntityID: 2, TargetEntityID: 1, Damage: 500, Penetrated: true, ResultCode: 3},
	}
	for i, hit := range hits {
		if hit != want[i] {
			t.Fatalf("hit %d = %+v, want %+v", i, hit, want[i])
		}
	}
}

// TestLiveHitsSkipUnusableMessages keeps malformed and self-directed envelopes
// out of the history.
func TestLiveHitsSkipUnusableMessages(t *testing.T) {
	var players livePlayers
	if err := players.event(headerEvent()); err != nil {
		t.Fatal(err)
	}
	short := FollowEvent{Event: StreamEvent{Kind: EventPacket, Packet: &Packet{
		Payload:    Payload{Kind: PayloadEntityMethod, EntityMethod: &EntityMethod{SubType: 9}},
		RawPayload: make([]byte, 17),
	}}}
	for _, event := range []FollowEvent{
		hitPacket(10, 0, 2, 300, 1, 3),
		hitPacket(11, 1, 0, 300, 1, 3),
		hitPacket(12, 3, 3, 300, 1, 3),
		short,
	} {
		if err := players.event(event); err != nil {
			t.Fatal(err)
		}
	}
	if hits := players.snapshotHits(); hits != nil {
		t.Fatalf("kept unusable hits: %+v", hits)
	}
}

// TestLiveHitsClearOnReset ties the history to the followed session.
func TestLiveHitsClearOnReset(t *testing.T) {
	var players livePlayers
	if err := players.event(headerEvent()); err != nil {
		t.Fatal(err)
	}
	if err := players.event(hitPacket(10, 1, 2, 300, 1, 3)); err != nil {
		t.Fatal(err)
	}
	if err := players.event(FollowEvent{Event: StreamEvent{Kind: EventReset, Reason: "replaced"}}); err != nil {
		t.Fatal(err)
	}
	if hits := players.snapshotHits(); hits != nil {
		t.Fatalf("history survived a session reset: %+v", hits)
	}
}

// TestLiveHitsBoundTheHistory keeps only the newest messages once the bound is
// reached, so a runaway recording cannot grow the snapshot.
func TestLiveHitsBoundTheHistory(t *testing.T) {
	var players livePlayers
	if err := players.event(headerEvent()); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < maxLiveHits+5; i++ {
		if err := players.event(hitPacket(float32(i), 1, 2, uint16(i+1), 1, 3)); err != nil {
			t.Fatal(err)
		}
	}
	hits := players.snapshotHits()
	if len(hits) != maxLiveHits {
		t.Fatalf("history holds %d hits, want %d", len(hits), maxLiveHits)
	}
	if hits[0].Clock != 5 || hits[len(hits)-1].Clock != float32(maxLiveHits+4) {
		t.Fatalf("kept the wrong window: first %+v last %+v", hits[0], hits[len(hits)-1])
	}
}
