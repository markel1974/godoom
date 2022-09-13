package quickhull

import "math"

const (
	minUint = uint(0)
	maxUint = ^minUint
	//minInt  = -maxInt - 1
	maxInt = int(maxUint >> 1)
)

func triangleNormal(a Vector, b Vector, c Vector) Vector {
	x := a.X - c.X
	y := a.Y - c.Y
	z := a.Z - c.Z
	rhsX := b.X - c.X
	rhsY := b.Y - c.Y
	rhsZ := b.Z - c.Z
	return Vector{
		X: y*rhsZ - z*rhsY,
		Y: z*rhsX - x*rhsZ,
		Z: x*rhsY - y*rhsX,
	}
}

func signedDistanceToPlane(v Vector, p plane) float64 {
	return p.n.Dot(v) + p.d
}

func squaredDistanceBetweenPointAndRay(p Vector, r ray) float64 {
	s := p.Sub(r.s)
	t := s.Dot(r.v)
	return s.Norm2() - t*t*r.vInvLengthSquared
}

// Find indices of extreme values (max x, min x, max y, min y, max z, min z) for the given point cloud
func extremeValues(vertexData []Vector) (extremeValueIndices [6]int) {
	vd0 := vertexData[0]
	extremeVals := [6]float64{vd0.X, vd0.X, vd0.Y, vd0.Y, vd0.Z, vd0.Z}

	xv := func(i, i2 int, v float64, idx int) {
		if v > extremeVals[i] {
			extremeVals[i] = v
			extremeValueIndices[i] = idx
		} else if v < extremeVals[i2] {
			extremeVals[i2] = v
			extremeValueIndices[i2] = idx
		}
	}

	for i, pos := range vertexData[1:] {
		idx := i + 1
		xv(0, 1, pos.X, idx)
		xv(2, 3, pos.Y, idx)
		xv(4, 5, pos.Z, idx)
	}

	return
}

type vectorField int

const (
	vfX vectorField = iota
	vfY
	vfZ
)

// Compute scale of the vertex data.
func scale(vertexData []Vector, extremeValueIndices [6]int) (s float64) {
	scl := func(i int, valType vectorField) {
		v := vertexData[extremeValueIndices[i]]
		var a float64
		switch valType {
		case vfX:
			a = v.X
		case vfY:
			a = v.Y
		case vfZ:
			a = v.Z
		}
		s = math.Max(s, math.Abs(a))
	}

	scl(0, vfX)
	scl(1, vfX)
	scl(2, vfY)
	scl(3, vfY)
	scl(4, vfZ)
	scl(5, vfZ)

	return
}
