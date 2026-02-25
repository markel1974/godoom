package quickhull

type ray struct {
	s                 Vector
	v                 Vector
	vInvLengthSquared float64
}

func newRay(s, v Vector) ray {
	return ray{s: s, v: v, vInvLengthSquared: 1 / v.Dot(v)}
}
