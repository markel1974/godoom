package lumps

import "github.com/markel1974/godoom/mr_tech/model/geometry"

type RawFace struct {
	Points  []geometry.XYZ
	TexName string
	IsSky   bool
}

// CreateXYZ creates and returns a geometry.XYZ struct using the provided x, y, and z coordinates.
func CreateXYZ(x, y, z float64) geometry.XYZ {
	// Conversione coordinate: Quake Z-up -> Engine Z-up
	//pos := geometry.XYZ{X: x, Y: z, Z: -y}
	pos := geometry.XYZ{X: x, Y: y, Z: z}
	return pos
}
