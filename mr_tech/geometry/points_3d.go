package geometry

import "math"

// XYZ represents a point or vector in 3D space with X, Y, and Z coordinates.
type XYZ struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// To32 converts the XYZ coordinates to single-precision XYZ32 values.
func (v *XYZ) To32() XYZ32 {
	return XYZ32{X: float32(v.X), Y: float32(v.Y), Z: float32(v.Z)}
}

// ToFloat64 returns the X, Y, and Z components as float64 values.
func (v *XYZ) ToFloat64() (float64, float64, float64) {
	return v.X, v.Y, v.Z
}

// Scale divides the X, Y and Z components of the XYZ struct by the given scale factor.
func (v *XYZ) Scale(scale XYZ) {
	v.X *= scale.X
	v.Y *= scale.Y
	v.Z *= scale.Z
}

// SubXYZ subtracts two geometry.XYZ vectors component-wise and returns the resulting vector.
func (v *XYZ) SubXYZ(b XYZ) XYZ {
	return XYZ{X: v.X - b.X, Y: v.Y - b.Y, Z: v.Z - b.Z}
}

// DotXYZ computes the dot product of two 3D vectors represented by geometry.XYZ structs.
func (v *XYZ) DotXYZ(b XYZ) float64 { return v.X*b.X + v.Y*b.Y + v.Z*b.Z }

// CrossXYZ computes the cross product of two 3D vectors a and b, returning the resulting vector as geometry.XYZ.
func (v *XYZ) CrossXYZ(b XYZ) XYZ {
	return XYZ{X: v.Y*b.Z - v.Z*b.Y, Y: v.Z*b.X - v.X*b.Z, Z: v.X*b.Y - v.Y*b.X}
}

// NormalizeXYZ normalizes a 3D vector to unit length and returns the resulting vector.
func (v *XYZ) NormalizeXYZ() XYZ {
	lenInv := 1.0 / math.Sqrt(v.X*v.X+v.Y*v.Y+v.Z*v.Z)
	return XYZ{X: v.X * lenInv, Y: v.Y * lenInv, Z: v.Z * lenInv}
}

// DistPointToSegmentSq returns the squared distance from p to the line segment from a to b.
// It treats a near-zero-length segment as a single point at a.
func DistPointToSegmentSq(p XYZ, a, b XYZ32) float64 {
	abX := float64(b.X) - float64(a.X)
	abY := float64(b.Y) - float64(a.Y)
	abZ := float64(b.Z) - float64(a.Z)
	apX := p.X - float64(a.X)
	apY := p.Y - float64(a.Y)
	apZ := p.Z - float64(a.Z)
	abLenSq := abX*abX + abY*abY + abZ*abZ
	if abLenSq <= 1e-18 {
		return apX*apX + apY*apY + apZ*apZ
	}
	t := (apX*abX + apY*abY + apZ*abZ) / abLenSq
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	cx := apX - abX*t
	cy := apY - abY*t
	cz := apZ - abZ*t
	return cx*cx + cy*cy + cz*cz
}

func DistPointToTriangleSq(p XYZ, a32, b32, c32 XYZ32) float64 {
	ax, ay, az := float64(a32.X), float64(a32.Y), float64(a32.Z)
	bx, by, bz := float64(b32.X), float64(b32.Y), float64(b32.Z)
	cx, cy, cz := float64(c32.X), float64(c32.Y), float64(c32.Z)

	abX, abY, abZ := bx-ax, by-ay, bz-az
	acX, acY, acZ := cx-ax, cy-ay, cz-az
	apX, apY, apZ := p.X-ax, p.Y-ay, p.Z-az

	d1 := abX*apX + abY*apY + abZ*apZ
	d2 := acX*apX + acY*apY + acZ*apZ
	if d1 <= 0 && d2 <= 0 {
		return apX*apX + apY*apY + apZ*apZ
	}

	bpX, bpY, bpZ := p.X-bx, p.Y-by, p.Z-bz
	d3 := abX*bpX + abY*bpY + abZ*bpZ
	d4 := acX*bpX + acY*bpY + acZ*bpZ
	if d3 >= 0 && d4 <= d3 {
		return bpX*bpX + bpY*bpY + bpZ*bpZ
	}

	vc := d1*d4 - d3*d2
	if vc <= 0 && d1 >= 0 && d3 <= 0 {
		if denom := d1 - d3; denom > 1e-18 {
			v := d1 / denom
			dx, dy, dz := apX-abX*v, apY-abY*v, apZ-abZ*v
			return dx*dx + dy*dy + dz*dz
		}
	}

	cpX, cpY, cpZ := p.X-cx, p.Y-cy, p.Z-cz
	d5 := abX*cpX + abY*cpY + abZ*cpZ
	d6 := acX*cpX + acY*cpY + acZ*cpZ
	if d6 >= 0 && d5 <= d6 {
		return cpX*cpX + cpY*cpY + cpZ*cpZ
	}

	vb := d5*d2 - d1*d6
	if vb <= 0 && d2 >= 0 && d6 <= 0 {
		if denom := d2 - d6; denom > 1e-18 {
			w := d2 / denom
			dx, dy, dz := apX-acX*w, apY-acY*w, apZ-acZ*w
			return dx*dx + dy*dy + dz*dz
		}
	}

	va := d3*d6 - d5*d4
	if va <= 0 && (d4-d3) >= 0 && (d5-d6) >= 0 {
		if denom := (d4 - d3) + (d5 - d6); denom > 1e-18 {
			w := (d4 - d3) / denom
			bcX, bcY, bcZ := cx-bx, cy-by, cz-bz
			dx, dy, dz := bpX-bcX*w, bpY-bcY*w, bpZ-bcZ*w
			return dx*dx + dy*dy + dz*dz
		}
	}

	denom := va + vb + vc
	if denom <= 1e-18 {
		dAB := DistPointToSegmentSq(p, a32, b32)
		dBC := DistPointToSegmentSq(p, b32, c32)
		dCA := DistPointToSegmentSq(p, c32, a32)
		if dBC < dAB {
			dAB = dBC
		}
		if dCA < dAB {
			dAB = dCA
		}
		return dAB
	}
	inv := 1.0 / denom
	v := vb * inv
	w := vc * inv
	dx := apX - abX*v - acX*w
	dy := apY - abY*v - acY*w
	dz := apZ - abZ*v - acZ*w
	return dx*dx + dy*dy + dz*dz
}
