package geometry

const Triangle3DPoints = 3

// Triangle3D represents a triangle in 3D space defined by three vertices.
type Triangle3D struct {
	Points [Triangle3DPoints]XYZ
}

// NewTriangle3D returns a new Triangle3D initialized with the provided points.
func NewTriangle3D(points [Triangle3DPoints]XYZ) *Triangle3D {
	return &Triangle3D{Points: points}
}

// Translate returns the coordinates of the vertex at idx shifted by the given X, Y, and Z offsets.
// It does not modify the triangle.
func (tri *Triangle3D) Translate(idx int, offX, offY, offZ float64) (float64, float64, float64) {
	p0x, p0y, p0z := tri.Points[idx].X+offX, tri.Points[idx].Y+offY, tri.Points[idx].Z+offZ
	return p0x, p0y, p0z
}

// PointInside2d reports whether the 2D point px, py lies inside the triangle's XY projection.
// Points on or near the triangle edges are treated as inside.
func (tri *Triangle3D) PointInside2d(px, py float64) bool {
	p0, p1, p2 := tri.Points[0], tri.Points[1], tri.Points[2]
	d1 := (px-p0.X)*(p1.Y-p0.Y) - (py-p0.Y)*(p1.X-p0.X)
	d2 := (px-p1.X)*(p2.Y-p1.Y) - (py-p1.Y)*(p2.X-p1.X)
	d3 := (px-p2.X)*(p0.Y-p2.Y) - (py-p2.Y)*(p0.X-p2.X)
	const eps = -0.001
	hasNeg := (d1 < eps) || (d2 < eps) || (d3 < eps)
	hasPos := (d1 > -eps) || (d2 > -eps) || (d3 > -eps)
	return !(hasNeg && hasPos)
}

// GetPoint0 returns the X, Y, and Z coordinates of the triangle's first point.
func (tri *Triangle3D) GetPoint0() (float64, float64, float64) {
	return tri.Points[0].X, tri.Points[0].Y, tri.Points[0].Z
}

// GetPoint1 returns the X, Y, and Z coordinates of the triangle's second point.
func (tri *Triangle3D) GetPoint1() (float64, float64, float64) {
	return tri.Points[1].X, tri.Points[1].Y, tri.Points[1].Z
}

// GetPoint2 returns the X, Y, and Z coordinates of the third triangle vertex.
func (tri *Triangle3D) GetPoint2() (float64, float64, float64) {
	return tri.Points[2].X, tri.Points[2].Y, tri.Points[2].Z
}

// Zero resets all triangle vertex coordinates to the origin.
func (tri *Triangle3D) Zero() {
	tri.Points[0].X = 0
	tri.Points[0].Y = 0
	tri.Points[0].Z = 0
	tri.Points[1].X = 0
	tri.Points[1].Y = 0
	tri.Points[1].Z = 0
	tri.Points[2].X = 0
	tri.Points[2].Y = 0
	tri.Points[2].Z = 0
}

// TransformPoints copies points from src into tri, offset by the given origin coordinates.
func (tri *Triangle3D) TransformPoints(src *Triangle3D, originX, originY, originZ float64) {
	tri.Points[0].X = src.Points[0].X + originX
	tri.Points[0].Y = src.Points[0].Y + originY
	tri.Points[0].Z = src.Points[0].Z + originZ
	tri.Points[1].X = src.Points[1].X + originX
	tri.Points[1].Y = src.Points[1].Y + originY
	tri.Points[1].Z = src.Points[1].Z + originZ
	tri.Points[2].X = src.Points[2].X + originX
	tri.Points[2].Y = src.Points[2].Y + originY
	tri.Points[2].Z = src.Points[2].Z + originZ
}

// CopyPoints copies all point coordinates from src into tri.
func (tri *Triangle3D) CopyPoints(src *Triangle3D) {
	tri.Points[0].X = src.Points[0].X
	tri.Points[0].Y = src.Points[0].Y
	tri.Points[0].Z = src.Points[0].Z
	tri.Points[1].X = src.Points[1].X
	tri.Points[1].Y = src.Points[1].Y
	tri.Points[1].Z = src.Points[1].Z
	tri.Points[2].X = src.Points[2].X
	tri.Points[2].Y = src.Points[2].Y
	tri.Points[2].Z = src.Points[2].Z
}

// MinMax returns the minimum and maximum X, Y, and Z coordinates across the triangle's three points.
func (tri *Triangle3D) MinMax() (minX, minY, minZ, maxX, maxY, maxZ float64) {
	pts := tri.Points
	minX, maxX = pts[0].X, pts[0].X
	minY, maxY = pts[0].Y, pts[0].Y
	minZ, maxZ = pts[0].Z, pts[0].Z

	for i := 1; i < Triangle3DPoints; i++ {
		if pts[i].X < minX {
			minX = pts[i].X
		} else if pts[i].X > maxX {
			maxX = pts[i].X
		}
		if pts[i].Y < minY {
			minY = pts[i].Y
		} else if pts[i].Y > maxY {
			maxY = pts[i].Y
		}
		if pts[i].Z < minZ {
			minZ = pts[i].Z
		} else if pts[i].Z > maxZ {
			maxZ = pts[i].Z
		}
	}
	return
}

// MinMaxExpands expands the supplied 3D bounds to include all triangle points and returns the updated bounds.
func (tri *Triangle3D) MinMaxExpands(minX, minY, minZ, maxX, maxY, maxZ float64) (float64, float64, float64, float64, float64, float64) {
	for _, p := range tri.Points {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
		if p.Z < minZ {
			minZ = p.Z
		}
		if p.Z > maxZ {
			maxZ = p.Z
		}
	}
	return minX, minY, minZ, maxX, maxY, maxZ
}

// SumPoints adds the triangle point coordinates to the provided totals and returns the updated sums and count.
func (tri *Triangle3D) SumPoints(cx, cy, cz float64, count int) (float64, float64, float64, int) {
	for _, p := range tri.Points {
		cx += p.X
		cy += p.Y
		cz += p.Z
		count++
	}
	return cx, cy, cz, count
}

func (tri *Triangle3D) TransformPointsWithMatrix(dst *Triangle3D, src *Triangle3D, originX, originY, originZ float64) {
	for k := 0; k < 3; k++ {
		rx := src.Points[k].X*tri.Points[0].X + src.Points[k].Y*tri.Points[1].X + src.Points[k].Z*tri.Points[2].X
		ry := src.Points[k].X*tri.Points[0].Y + src.Points[k].Y*tri.Points[1].Y + src.Points[k].Z*tri.Points[2].Y
		rz := src.Points[k].X*tri.Points[0].Z + src.Points[k].Y*tri.Points[1].Z + src.Points[k].Z*tri.Points[2].Z

		dst.Points[k].X = rx + originX
		dst.Points[k].Y = ry + originY
		dst.Points[k].Z = rz + originZ
	}
}

// RotatePoint rotates a point (pX, pY, pZ) based on the TagAxis transformation matrix and returns the transformed coordinates.
func (tri *Triangle3D) RotatePoint(pX, pY, pZ float64) (float64, float64, float64) {
	x := pX*tri.Points[0].X + pY*tri.Points[1].X + pZ*tri.Points[2].X
	y := pX*tri.Points[0].Y + pY*tri.Points[1].Y + pZ*tri.Points[2].Y
	z := pX*tri.Points[0].Z + pY*tri.Points[1].Z + pZ*tri.Points[2].Z
	return x, y, z
}

// RotatePointInverse transforms a vector (pX, pY, pZ) by the transpose (inverse) of the TagAxis 3x3 basis matrix.
func (tri *Triangle3D) RotatePointInverse(pX, pY, pZ float64) (float64, float64, float64) {
	x := pX*tri.Points[0].X + pY*tri.Points[0].Y + pZ*tri.Points[0].Z
	y := pX*tri.Points[1].X + pY*tri.Points[1].Y + pZ*tri.Points[1].Z
	z := pX*tri.Points[2].X + pY*tri.Points[2].Y + pZ*tri.Points[2].Z
	return x, y, z
}

func (tri *Triangle3D) IsEmpty() bool {
	if tri.Points[0].X == 0 && tri.Points[0].Y == 0 && tri.Points[0].Z == 0 &&
		tri.Points[1].X == 0 && tri.Points[1].Y == 0 && tri.Points[1].Z == 0 &&
		tri.Points[2].X == 0 && tri.Points[2].Y == 0 && tri.Points[2].Z == 0 {
		return true
	}
	return false
}

// FindDeepestPoint returns the coordinates of the triangle vertex with the smallest dot product along dir.
func (tri *Triangle3D) FindDeepestPoint(dir XYZ) (float64, float64, float64) {
	bestP := dir.DotXYZ(tri.Points[0])
	bestIdx := 0
	for i := 1; i < 3; i++ {
		p := dir.DotXYZ(tri.Points[i])
		if p < bestP {
			bestP = p
			bestIdx = i
		}
	}
	return tri.Points[bestIdx].X, tri.Points[bestIdx].Y, tri.Points[bestIdx].Z
}

// Project returns the minimum and maximum dot-product values of the triangle's points along axis.
func (tri *Triangle3D) Project(axis XYZ) (float64, float64) {
	minP := axis.DotXYZ(tri.Points[0])
	maxP := minP
	for i := 1; i < 3; i++ {
		p := axis.DotXYZ(tri.Points[i])
		if p < minP {
			minP = p
		}
		if p > maxP {
			maxP = p
		}
	}
	return minP, maxP
}
