package geometry

import "math"

// Triangulate3DFace decomposes a 3D polygon into a set of non-overlapping triangles with 3D coordinates.
func Triangulate3DFace(pts []XYZ) [][]XYZ {
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
			tri3d[i] = resolve3DPoint(t2d[i], pts, nx, ny, nz, axis)
		}
		output = append(output, tri3d)
	}

	return output
}

// resolve3DPoint calculates the 3D coordinates of a point on a plane using its 2D projection and the plane's equation.
// It preserves exact values if the point matches existing original points, avoiding floating-point drift over iterations.
func resolve3DPoint(p2d XY, originalPts []XYZ, nx, ny, nz float64, axis int) XYZ {
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
