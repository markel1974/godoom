package wad

// Point represents a coordinate in a 2D space with X and Y integer components.
type Point struct {
	X int16
	Y int16
}

// XY represents a point in a 2D space with X and Y coordinates as floating-point numbers.
type XY struct {
	X float64
	Y float64
}

// Point3 represents a point in a 3-dimensional space with additional U and V coordinates.
type Point3 struct {
	X float64
	Y float64
	Z float64
	U float64
	V float64
}

// MakePoint3F creates a Point3 struct with the specified float64 values for X, Y, Z, U, and V.
func MakePoint3F(x, y, z, u, v float64) Point3 {
	return Point3{X: x, Y: y, Z: z, U: u, V: v}
}

// MakePoint3 creates a Point3 instance from int16 values by converting them to float64 and delegating to MakePoint3F.
func MakePoint3(x, y, z, u, v int16) Point3 {
	return MakePoint3F(float64(x), float64(y), float64(z), float64(u), float64(v))
}
