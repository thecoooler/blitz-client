package replay

import (
	"encoding/json"
	"testing"
)

func TestLiveModeUnknownZeroDetachedAndReset(t *testing.T) {
	var players livePlayers
	apply := func(args *BasePlayerCreate) {
		t.Helper()
		packet := &Packet{Payload: Payload{Kind: PayloadBasePlayerCreate, AuthorNickname: "author", ArenaUniqueID: 42, ArenaTypeID: 20, Arguments: args}}
		if err := players.event(FollowEvent{Event: StreamEvent{Kind: EventPacket, Packet: packet}}); err != nil {
			t.Fatal(err)
		}
	}
	snapshot := func() LiveReplayState {
		state := LiveReplayState{}
		(&LiveReplayWatcher{}).enrichLiveState(&state, &liveRecording{players: players})
		return state
	}
	apply(nil)
	if snapshot().Mode != nil {
		t.Fatal("missing context became a known mode")
	}
	apply(&BasePlayerCreate{})
	state := snapshot()
	if state.Mode == nil || *state.Mode != (LiveBattleMode{}) {
		t.Fatalf("valid zero mode lost: %+v", state.Mode)
	}
	data, err := json.Marshal(state.Mode)
	if err != nil || string(data) != `{"battle_category_id":0,"matchmaker_type":0}` {
		t.Fatalf("zero mode JSON: %s %v", data, err)
	}
	state.Mode.MatchmakerType = 9
	if snapshot().Mode.MatchmakerType != 0 {
		t.Fatal("mode snapshot mutated decoder")
	}
	apply(nil)
	if snapshot().Mode != nil {
		t.Fatal("old mode survived new context without arguments")
	}
	apply(&BasePlayerCreate{BattleCategoryID: 3, MatchmakerType: 2})
	if err := players.event(FollowEvent{Event: StreamEvent{Kind: EventReset}}); err != nil {
		t.Fatal(err)
	}
	if snapshot().Mode != nil || snapshot().Battle != nil {
		t.Fatal("mode survived replay reset")
	}
}
