package jsonx

import (
	"math"
	"testing"
)

// Expected strings were produced by serde_json 1.0 from the Rust crate.
func TestFormatsLikeSerdeJSON(t *testing.T) {
	f32s := []float32{0.1, 1.0, 1e20, 1e-7, 123456789.0, 1e13, 1e-6, 1.5e-5, float32(math.Copysign(0, -1))}
	expected32 := `[0.1,1.0,1e+20,1e-7,123456790.0,1e+13,0.000001,0.000015,-0.0]`
	if got, _ := Marshal(F32s(f32s)); string(got) != expected32 {
		t.Fatal(string(got))
	}
	widened := `[0.10000000149011612,1.0,1.0000000200408773e+20,1.0000000116860974e-7,123456792.0,9999999827968.0,9.999999974752427e-7,0.000014999999621068127,-0.0]`
	if got, _ := Marshal(Widen(f32s)); string(got) != widened {
		t.Fatal(string(got))
	}
	a, b := 0.1, 0.2
	f64s := []F64{1.0, 1e16, 1e17, 1e-5, 1e-4, 1.5e300, 123.0e15, F64(a + b), F64(math.NaN())}
	if got, _ := Marshal(f64s); string(got) != `[1.0,1e+16,1e+17,0.00001,0.0001,1.5e+300,1.23e+17,0.30000000000000004,null]` {
		t.Fatal(string(got))
	}
}
