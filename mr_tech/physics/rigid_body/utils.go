package rigid_body

// ITriangle represents a triangle defined by three points in 3D space.
type ITriangle interface {
	GetPoint0() (float64, float64, float64)
	GetPoint1() (float64, float64, float64)
	GetPoint2() (float64, float64, float64)
}

// CompileMassFinalize assumes volume6 is nonzero and moments are relative to the reference point.
func CompileMassFinalize(volume6 float64, refX, refY, refZ float64, mx, my, mz float64, sxx, sxy, sxz float64, syy, syz, szz float64) (float64, float64, float64, float64, float64, float64, float64, float64, float64) {
	// Centroid relative to the reference point.
	rx := mx / volume6
	ry := my / volume6
	rz := mz / volume6

	cx := refX + rx
	cy := refY + ry
	cz := refZ + rz

	// Signed volume.
	volume := volume6 / 6.0

	// Translate second moments to the centroid.
	sxx -= volume * rx * rx
	sxy -= volume * rx * ry
	sxz -= volume * rx * rz

	syy -= volume * ry * ry
	syz -= volume * ry * rz

	szz -= volume * rz * rz

	// Inertia tensor per unit mass:
	//
	// I = (trace(S) * Identity - S) / volume

	factor := 6.0 / volume6

	ixx := (syy + szz) * factor
	iyy := (sxx + szz) * factor
	izz := (sxx + syy) * factor

	ixy := -sxy * factor
	ixz := -sxz * factor
	iyz := -syz * factor

	return cx, cy, cz, ixx, ixy, ixz, iyy, iyz, izz
}

// CompileMassAccumulate updates accumulated signed volume and moment terms for one triangular face.
// It uses the supplied reference point to form a tetrahedron and returns the updated totals.
func CompileMassAccumulate(refX, refY, refZ float64, face ITriangle, volume6, mx, my, mz float64, sxx, sxy, sxz float64, syy, syz, szz float64) (float64, float64, float64, float64, float64, float64, float64, float64, float64, float64) {
	x0, y0, z0 := face.GetPoint0()
	x1, y1, z1 := face.GetPoint1()
	x2, y2, z2 := face.GetPoint2()

	// Vertices relative to the reference point.
	ax, ay, az := x0-refX, y0-refY, z0-refZ
	bx, by, bz := x1-refX, y1-refY, z1-refZ
	dx, dy, dz := x2-refX, y2-refY, z2-refZ

	// Six times the signed tetrahedron volume.
	v6 := ax*(by*dz-bz*dy) + ay*(bz*dx-bx*dz) + az*(bx*dy-by*dx)
	volume6 += v6

	// Sum of the three non-reference vertices.
	sx := ax + bx + dx
	sy := ay + by + dy
	sz := az + bz + dz

	// First moments.
	mx += v6 * sx * 0.25
	my += v6 * sy * 0.25
	mz += v6 * sz * 0.25

	// Second moments:
	// S = V/20 * (sum(v) * sum(v)^T + sum(v * v^T))
	// Since V = v6/6, the coefficient is v6/120.

	k := v6 / 120.0

	sxx += k * (sx*sx + ax*ax + bx*bx + dx*dx)
	sxy += k * (sx*sy + ax*ay + bx*by + dx*dy)
	sxz += k * (sx*sz + ax*az + bx*bz + dx*dz)

	syy += k * (sy*sy + ay*ay + by*by + dy*dy)
	syz += k * (sy*sz + ay*az + by*bz + dy*dz)

	szz += k * (sz*sz + az*az + bz*bz + dz*dz)

	return volume6, mx, my, mz, sxx, sxy, sxz, syy, syz, szz
}
