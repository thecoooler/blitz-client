package export

import "math"

// Products are converted explicitly to float32 so the compiler never fuses
// them into FMA instructions; results stay bit-identical to the Rust exporter.

type vec3 = [3]float32
type vec2 = [2]float32

func mul(a, b float32) float32 { return float32(a * b) }

func add3(a, b vec3) vec3 { return vec3{a[0] + b[0], a[1] + b[1], a[2] + b[2]} }
func sub3(a, b vec3) vec3 { return vec3{a[0] - b[0], a[1] - b[1], a[2] - b[2]} }
func scale3(v vec3, s float32) vec3 {
	return vec3{mul(v[0], s), mul(v[1], s), mul(v[2], s)}
}
func dot3(a, b vec3) float32 {
	return mul(a[0], b[0]) + mul(a[1], b[1]) + mul(a[2], b[2])
}
func cross3(a, b vec3) vec3 {
	return vec3{
		mul(a[1], b[2]) - mul(a[2], b[1]),
		mul(a[2], b[0]) - mul(a[0], b[2]),
		mul(a[0], b[1]) - mul(a[1], b[0]),
	}
}
func sqrt32(v float32) float32 { return float32(math.Sqrt(float64(v))) }
func length3(v vec3) float32   { return sqrt32(dot3(v, v)) }
func finite32(v float32) bool  { return !math.IsNaN(float64(v)) && !math.IsInf(float64(v), 0) }
func abs32(v float32) float32  { return float32(math.Abs(float64(v))) }
func recip(v float32) float32  { return 1 / v }
func allFinite(values ...float32) bool {
	for _, value := range values {
		if !finite32(value) {
			return false
		}
	}
	return true
}

// min32 and max32 ignore NaN like Rust's f32::min and f32::max.
func min32(a, b float32) float32 {
	if a != a || b < a {
		return b
	}
	return a
}
func max32(a, b float32) float32 {
	if a != a || b > a {
		return b
	}
	return a
}
