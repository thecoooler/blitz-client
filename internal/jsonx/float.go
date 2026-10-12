package jsonx

import (
	"math"
	"strconv"
	"strings"
)

// F32 is a float32 that encodes like serde_json's f32 serialization: shortest
// round-trip digits, integral values as "1.0", and NaN/Inf as null.
type F32 float32

// F64 is a float64 that encodes like serde_json's f64 serialization. Values
// that serde_json converted from f32 through serde_json::Value use F64 too.
type F64 float64

func (f F32) MarshalJSON() ([]byte, error) {
	return []byte(FormatFloat(float64(f), 32)), nil
}

func (f F64) MarshalJSON() ([]byte, error) {
	return []byte(FormatFloat(float64(f), 64)), nil
}

// F32s converts float32 values to F32.
func F32s(values []float32) []F32 {
	out := make([]F32, len(values))
	for index, value := range values {
		out[index] = F32(value)
	}
	return out
}

// Widen converts float32 values to F64, as serde_json::Value does.
func Widen(values []float32) []F64 {
	out := make([]F64, len(values))
	for index, value := range values {
		out[index] = F64(value)
	}
	return out
}

// FormatFloat reproduces the shortest round-trip formatting used by
// serde_json, which writes exponents with an explicit plus sign.
func FormatFloat(value float64, bits int) string {
	return formatFloat(value, bits, true)
}

// FormatRyu reproduces the ryu crate's formatting (used by serde_yaml_ng),
// which omits the plus sign in exponents. Callers handle NaN and infinities.
func FormatRyu(value float64, bits int) string {
	return formatFloat(value, bits, false)
}

func formatFloat(value float64, bits int, plusExponent bool) string {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "null"
	}
	scientific := strconv.FormatFloat(value, 'e', -1, bits)
	negative := strings.HasPrefix(scientific, "-")
	scientific = strings.TrimPrefix(scientific, "-")
	mantissa, exponentText, _ := strings.Cut(scientific, "e")
	digits := strings.Replace(mantissa, ".", "", 1)
	exponent, _ := strconv.Atoi(exponentText)
	length := len(digits)
	// value = digits * 10^k, and 10^(kk-1) <= value < 10^kk.
	k := exponent - (length - 1)
	kk := length + k
	maxDecimal, minDecimal := 16, -5
	if bits == 32 {
		maxDecimal, minDecimal = 13, -6
	}
	var out strings.Builder
	if negative {
		out.WriteByte('-')
	}
	switch {
	case value == 0:
		out.WriteString("0.0")
	case 0 <= k && kk <= maxDecimal:
		out.WriteString(digits)
		out.WriteString(strings.Repeat("0", k))
		out.WriteString(".0")
	case 0 < kk && kk <= maxDecimal:
		out.WriteString(digits[:kk])
		out.WriteByte('.')
		out.WriteString(digits[kk:])
	case minDecimal < kk && kk <= 0:
		out.WriteString("0.")
		out.WriteString(strings.Repeat("0", -kk))
		out.WriteString(digits)
	default:
		out.WriteByte(digits[0])
		if length > 1 {
			out.WriteByte('.')
			out.WriteString(digits[1:])
		}
		out.WriteByte('e')
		if kk-1 >= 0 && plusExponent {
			out.WriteByte('+')
		}
		out.WriteString(strconv.Itoa(kk - 1))
	}
	return out.String()
}
