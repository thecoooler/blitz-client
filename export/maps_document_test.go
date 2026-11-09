package export

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/thecoooler/blitz-client/client"
)

func TestRawMapsProjectionResolvesSourceAliases(t *testing.T) {
	document, err := client.DecodeMapsDocument([]byte(`maps:
    first:
        id: 1
        localName: FIRST
        frame: 3
        availableInTrainingRoom: true
        availableModes: [2, 4]
        assaultRespawnPoints:
            allies:
                - points: [[1.5, -2]]
                  respawnNumber: 0
`))
	if err != nil {
		t.Fatal(err)
	}
	raw := rawMapsFromDocument(document)
	first := raw.Maps["first"]
	if first.SpriteFrame == nil || *first.SpriteFrame != 3 {
		t.Fatalf("frame alias: %+v", first)
	}
	if first.AvaliableInTrainingRoom == nil || !*first.AvaliableInTrainingRoom {
		t.Fatalf("training-room alias: %+v", first)
	}
	if !reflect.DeepEqual(first.AvailableModes, []int64{2, 4}) {
		t.Fatalf("modes: %+v", first)
	}
	if first.AssaultRespawnPoints == nil || len(first.AssaultRespawnPoints.Allies) != 1 ||
		first.AssaultRespawnPoints.Allies[0].Points[0] != [2]float64{1.5, -2} {
		t.Fatalf("respawn points: %+v", first.AssaultRespawnPoints)
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]map[string]map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["maps"]["first"]["avaliableInTrainingRoom"] != true || decoded["maps"]["first"]["spriteFrame"] != float64(3) {
		t.Fatalf("wire shape: %s", encoded)
	}
}
