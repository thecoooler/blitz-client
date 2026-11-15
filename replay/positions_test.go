package replay

import (
	"archive/zip"
	"encoding/binary"
	"io"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// transformPacket builds the verified 49-byte vehicle transform envelope.
func transformPacket(entityID uint32, clock float32, x, y, z float32) *Packet {
	raw := make([]byte, vehicleTransformSize)
	binary.LittleEndian.PutUint32(raw, entityID)
	binary.LittleEndian.PutUint32(raw[4:], 7)
	binary.LittleEndian.PutUint32(raw[8:], 0)
	for axis, value := range [3]float32{x, y, z} {
		binary.LittleEndian.PutUint32(raw[12+axis*4:], math.Float32bits(value))
	}
	return &Packet{ClockSecs: clock, Payload: Payload{Kind: PayloadUnknown, PacketType: vehicleTransformType}, RawPayload: raw}
}

func TestVehiclePositionsFollowVerifiedLayout(t *testing.T) {
	var decoder vehiclePositionDecoder
	decoder.onPacket(transformPacket(11, 3.5, 10, 4, -20), "26.10.1_ruby_apple")
	decoder.onPacket(transformPacket(12, 4.5, -5, 4, 30), "26.10.1_ruby_apple")
	// A later report replaces the earlier one for the same vehicle.
	decoder.onPacket(transformPacket(11, 6, 14, 5, -22), "26.10.1_ruby_apple")
	if len(decoder.positions) != 2 {
		t.Fatalf("retained %d vehicles", len(decoder.positions))
	}
	position := decoder.positions[11]
	if position.X != 14 || position.Y != 5 || position.Z != -22 || position.Clock != 6 || position.EntityID != 11 {
		t.Fatalf("last report not retained: %+v", position)
	}

	// Unverified versions keep their packets raw.
	var unknown vehiclePositionDecoder
	unknown.onPacket(transformPacket(11, 1, 1, 1, 1), "26.10.2_ruby_apple")
	if len(unknown.positions) != 0 {
		t.Fatal("unverified version decoded")
	}

	// A packet of another length or type is not a transform.
	var wrong vehiclePositionDecoder
	short := transformPacket(11, 1, 1, 1, 1)
	short.RawPayload = short.RawPayload[:40]
	wrong.onPacket(short, "26.10.1_ruby_apple")
	other := transformPacket(11, 1, 1, 1, 1)
	other.Payload.PacketType = 39
	wrong.onPacket(other, "26.10.1_ruby_apple")
	if len(wrong.positions) != 0 {
		t.Fatal("mismatched packet decoded")
	}

	// A non-finite coordinate is not a position.
	var broken vehiclePositionDecoder
	invalid := transformPacket(11, 1, 1, 1, 1)
	binary.LittleEndian.PutUint32(invalid.RawPayload[16:], math.Float32bits(float32(math.NaN())))
	broken.onPacket(invalid, "26.10.1_ruby_apple")
	if len(broken.positions) != 0 {
		t.Fatal("non-finite coordinate decoded")
	}
}

func TestLivePositionsKeepRosterVehiclesOnly(t *testing.T) {
	ownID, enemyID := uint32(11), uint32(22)
	unknownID := uint32(99)
	p := livePlayers{author: "Author", version: "26.10.1_ruby_apple", players: map[uint32]ArenaPlayer{
		1: {AccountID: 1, Nickname: "Author", TeamNumber: 2, VehicleEntityID: &ownID},
		2: {AccountID: 2, Nickname: "Enemy", TeamNumber: 1, VehicleEntityID: &enemyID},
	}}
	feed := func(packet *Packet) {
		t.Helper()
		if err := p.event(FollowEvent{Event: StreamEvent{Kind: EventPacket, Packet: packet}}); err != nil {
			t.Fatal(err)
		}
	}
	feed(transformPacket(enemyID, 2, 5, 1, 7))
	// The author's separate player entity tracks the same tank; it is not a
	// roster vehicle, so no team could be shown for it.
	feed(transformPacket(unknownID, 2, 5, 1, 7))
	feed(transformPacket(ownID, 3, -5, 1, -7))
	roster, _ := p.snapshot("")
	if roster == nil {
		t.Fatal("roster missing")
	}
	positions := p.positions.snapshot(roster)
	if len(positions) != 2 || positions[0].EntityID != ownID || positions[1].EntityID != enemyID {
		t.Fatalf("positions: %+v", positions)
	}
	// The snapshot is detached from the decoder's retained state.
	positions[0].X = 1000
	if p.positions.positions[ownID].X != -5 {
		t.Fatal("snapshot shared decoder state")
	}
	if err := p.event(FollowEvent{Event: StreamEvent{Kind: EventReset}}); err != nil {
		t.Fatal(err)
	}
	if len(p.positions.positions) != 0 {
		t.Fatal("reset retained positions")
	}
}

func TestLivePositionsFromRecording(t *testing.T) {
	z, err := zip.OpenReader("../testdata/sample.tbreplay")
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	stream, err := z.Open("data.replay")
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(stream)
	stream.Close()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	directory := filepath.Join(root, "recording_iCooLER.tbreplay")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "data.replay"), data, 0600); err != nil {
		t.Fatal(err)
	}
	watcher := NewLiveReplayWatcher(root)
	defer watcher.Close()
	state, err := watcher.Poll()
	if err != nil {
		t.Fatal(err)
	}
	if state.Kind != "battle" || state.Roster == nil || len(state.Positions) == 0 {
		t.Fatalf("state: kind=%q positions=%d", state.Kind, len(state.Positions))
	}
	if !SupportsVehiclePositions(state.Diagnostics.ClientVersion) {
		t.Fatalf("fixture version: %q", state.Diagnostics.ClientVersion)
	}
	vehicles := map[uint32]bool{}
	for _, player := range append(append([]ArenaPlayer{}, state.Roster.Allies...), state.Roster.Enemies...) {
		if player.VehicleEntityID != nil {
			vehicles[*player.VehicleEntityID] = true
		}
	}
	for i, position := range state.Positions {
		if !vehicles[position.EntityID] || position.Clock <= 0 || math.Abs(float64(position.X)) > 1000 || math.Abs(float64(position.Z)) > 1000 {
			t.Fatalf("implausible position %d: %+v", i, position)
		}
		if i > 0 && state.Positions[i-1].EntityID >= position.EntityID {
			t.Fatalf("positions not sorted: %+v", state.Positions)
		}
	}
}
