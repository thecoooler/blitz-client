package archive

import (
	"testing"

	ka "github.com/thecoooler/blitz-client/internal/katest"
)

func TestParseKeyedArchiveDecodesDeviceSettings(t *testing.T) {
	data := ka.Archive(ka.Fields{
		{Key: "UseAutoLogin", Value: ka.Number(1)},
		{Key: "InternalWebToken2", Value: ka.Text("91803618:1:2")},
	})
	value, err := ParseKeyedArchive(data)
	if err != nil {
		t.Fatal(err)
	}
	if value.Kind != Map || value.Map["InternalWebToken2"].String != "91803618:1:2" || value.Map["UseAutoLogin"].Unsigned != 1 {
		t.Fatalf("decoded %+v", value)
	}
	if _, err := ParseKeyedArchive(append(data, 0)); err == nil {
		t.Fatal("trailing bytes were ignored")
	}
	if _, err := ParseKeyedArchive([]byte("SFV2")); err == nil {
		t.Fatal("non-archive input was accepted")
	}
}
