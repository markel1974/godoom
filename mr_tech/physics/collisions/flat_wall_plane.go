package collisions

import "math"

// FlatWallPlane represents a flat wall plane defined by a normal direction and a point on the plane.
type FlatWallPlane struct {
	nX, nY, nZ float64
	pX, pY, pZ float64
}

// NewFlatWallPlane creates a new flat wall plane with zeroed normal and point coordinates.
func NewFlatWallPlane() *FlatWallPlane {
	return &FlatWallPlane{}
}

// Assign sets the plane's normal vector and reference point coordinates.
func (fw *FlatWallPlane) Assign(nx, ny, nz float64, px, py, pz float64) {
	fw.nX, fw.nY, fw.nZ = nx, ny, nz
	fw.pX, fw.pY, fw.pZ = px, py, pz
}

// IsCoplanar reports whether the given normal and point define the same plane within tolerance.
func (fw *FlatWallPlane) IsCoplanar(nx, ny, nz float64, ax, ay, az float64) bool {
	if nx*fw.nX+ny*fw.nY+nz*fw.nZ > 0.999 && math.Abs((ax-fw.pX)*fw.nX+(ay-fw.pY)*fw.nY+(az-fw.pZ)*fw.nZ) < planeEpsilon {
		return true
	}
	return false
}
