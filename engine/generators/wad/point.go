package wad

type Point struct {
	X int16
	Y int16
}

type XY struct {
	X float64
	Y float64
}

type Point3 struct {
	X float64
	Y float64
	Z float64
	U float64
	V float64
}

func MakePoint3F(x, y, z, u, v float64) Point3 {
	return Point3{X: x, Y: y, Z: z, U: u, V: v}
}

func MakePoint3(x, y, z, u, v int16) Point3 {
	return MakePoint3F(float64(x), float64(y), float64(z), float64(u), float64(v))
}
