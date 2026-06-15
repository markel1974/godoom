package geometry

import "math"

// Triangulate3d decomposes a 3D polygon into a set of non-overlapping triangles with 3D coordinates.
func Triangulate3d(pts []XYZ) [][]XYZ {
	pLen := len(pts)
	if pLen == 3 {
		return [][]XYZ{{pts[0], pts[1], pts[2]}}
	}

	// 1. Calculate polygon normal (Newell's Method)
	var nx, ny, nz float64
	for i := 0; i < pLen; i++ {
		curr := pts[i]
		next := pts[(i+1)%pLen]
		nx += (curr.Y - next.Y) * (curr.Z + next.Z)
		ny += (curr.Z - next.Z) * (curr.X + next.X)
		nz += (curr.X - next.X) * (curr.Y + next.Y)
	}

	// 2. Find the dominant projection plane to maximize 2D area
	absX, absY, absZ := math.Abs(nx), math.Abs(ny), math.Abs(nz)
	axis := 2 // 0: drop X, 1: drop Y, 2: drop Z
	if absX >= absY && absX >= absZ {
		axis = 0
	} else if absY >= absX && absY >= absZ {
		axis = 1
	}

	// 3. Project onto 2D plane using the native Polygon type
	poly2d := make(Polygon, pLen)
	for i, p := range pts {
		switch axis {
		case 0:
			poly2d[i] = XY{X: p.Y, Y: p.Z}
		case 1:
			poly2d[i] = XY{X: p.X, Y: p.Z}
		default:
			poly2d[i] = XY{X: p.X, Y: p.Y}
		}
	}

	// 4. Delegate to CDT engine (Bowyer-Watson + PSLG)
	mesh2d := poly2d.Triangulate()

	// 5. Un-project: Reconstruct the 3D mesh
	var output [][]XYZ
	for _, t2d := range mesh2d {
		if len(t2d) != 3 {
			continue
		}
		tri3d := make([]XYZ, 3)
		for i := 0; i < 3; i++ {
			tri3d[i] = resolve3dPoint(t2d[i], pts, nx, ny, nz, axis)
		}
		output = append(output, tri3d)
	}

	return output
}

// resolve3DPoint calculates the 3D coordinates of a point on a plane using its 2D projection and the plane's equation.
// It preserves exact values if the point matches existing original points, avoiding floating-point drift over iterations.
func resolve3dPoint(p2d XY, originalPts []XYZ, nx, ny, nz float64, axis int) XYZ {
	// Phase A: Check against original vertices (Preserves exact floats and avoids numerical drift)
	for _, op := range originalPts {
		var u, v float64
		switch axis {
		case 0:
			u, v = op.Y, op.Z
		case 1:
			u, v = op.X, op.Z
		default:
			u, v = op.X, op.Y
		}
		// Tolerance for identification (epsilon)
		if math.Abs(u-p2d.X) < 0.0001 && math.Abs(v-p2d.Y) < 0.0001 {
			return op
		}
	}

	// Phase B: If the PSLG engine has injected a new vertex (e.g., T-Junction),
	// calculate the missing orthogonal coordinate using the plane equation (Ax + By + Cz = D)
	ref := originalPts[0]
	d := nx*ref.X + ny*ref.Y + nz*ref.Z

	switch axis {
	case 0: // Y, Z known. Find X
		x := (d - ny*p2d.X - nz*p2d.Y) / nx
		return XYZ{X: x, Y: p2d.X, Z: p2d.Y}
	case 1: // X, Z known. Find Y
		y := (d - nx*p2d.X - nz*p2d.Y) / ny
		return XYZ{X: p2d.X, Y: y, Z: p2d.Y}
	default: // X, Y known. Find Z
		z := (d - nx*p2d.X - ny*p2d.Y) / nz
		return XYZ{X: p2d.X, Y: p2d.Y, Z: z}
	}
}

// ClosestPointSegmentSegment calculates the closest points between two 3D line segments defined by their points and directions.
// p1 and d1 define the starting point and direction vector of the first segment, respectively.
// p2 and d2 define the starting point and direction vector of the second segment, respectively.
// Returns the closest points on the two segments as two geometry.XYZ instances.
func ClosestPointSegmentSegment(p1, d1, p2, d2 XYZ) (XYZ, XYZ) {
	r := SubXYZ(p1, p2)
	a, e, f := DotXYZ(d1, d1), DotXYZ(d2, d2), DotXYZ(d2, r)
	c := DotXYZ(d1, r)
	b := DotXYZ(d1, d2)
	denom := a*e - b*b
	s, t := 0.0, 0.0
	if denom != 0.0 {
		s = max(0.0, min(1.0, (b*f-c*e)/denom))
	}
	t = (b*s + f) / e
	t = max(0.0, min(1.0, t))
	s = (b*t - c) / a
	s = max(0.0, min(1.0, s))
	c1 := XYZ{X: p1.X + d1.X*s, Y: p1.Y + d1.Y*s, Z: p1.Z + d1.Z*s}
	c2 := XYZ{X: p2.X + d2.X*t, Y: p2.Y + d2.Y*t, Z: p2.Z + d2.Z*t}
	return c1, c2
}

// TriangleProject projects a triangle's vertices onto a given axis and returns the minimum and maximum projection values.
func TriangleProject(tri [3]XYZ, axis XYZ) (float64, float64) {
	minP := DotXYZ(axis, tri[0])
	maxP := minP
	for i := 1; i < 3; i++ {
		p := DotXYZ(axis, tri[i])
		if p < minP {
			minP = p
		}
		if p > maxP {
			maxP = p
		}
	}
	return minP, maxP
}

// TriangleFindDeepestPoint computes the vertex in a triangle with the smallest projection onto a given direction vector.
// dir is the reference direction for projection.
// v is an array of three 3D points representing the vertices of a triangle.
// Returns the X, Y, and Z coordinates of the vertex with the smallest projection.
func TriangleFindDeepestPoint(tri [3]XYZ, dir XYZ) (float64, float64, float64) {
	bestP := DotXYZ(dir, tri[0])
	bestIdx := 0
	for i := 1; i < 3; i++ {
		p := DotXYZ(dir, tri[i])
		if p < bestP {
			bestP = p
			bestIdx = i
		}
	}
	return tri[bestIdx].X, tri[bestIdx].Y, tri[bestIdx].Z
}
