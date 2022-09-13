package quickhull


type plane struct {
	n          Vector
	d          float64 // Signed distance (if normal is of length 1) to the plane from origin
	sqrNLength float64 // Normal length squared
}

func (p plane) isPointOnPositiveSide(q Vector) bool {
	return p.n.Dot(q)+p.d >= 0
}

func newPlane(n Vector, p Vector) plane {
	return plane{n: n, d: -n.Dot(p), sqrNLength: n.Dot(n)}
}
