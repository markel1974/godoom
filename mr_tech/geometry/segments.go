package geometry

const segmentsTolerance = 1e-6 // segmentsTolerance defines the precision threshold for geometric calculations, such as collinearity and point-segment checks.

// IsSegmentSubset determines if the segment defined by points a1 and a2 is a subset of the segment defined by points b1 and b2.
func IsSegmentSubset(a1, a2, b1, b2 XY) bool {
	// Collinearity Condition: A1 and A2 must lie on the infinite line passing through B1 and B2.
	if !isCollinear(b1, b2, a1) || !isCollinear(b1, b2, a2) {
		return false
	}
	// Inclusion Condition: Both points A1 and A2 must be located "between" B1 and B2.
	return isPointOnSegment(b1, b2, a1) && isPointOnSegment(b1, b2, a2)
}

// isCollinear determines if a point p is collinear with a line segment defined by points s1 and s2, using a tolerance.
func isCollinear(s1, s2, p XY) bool {
	dx := s2.X - s1.X
	dy := s2.Y - s1.Y
	lengthSq := dx*dx + dy*dy

	if lengthSq == 0 {
		return DistanceSq(s1, p) < segmentsTolerance*segmentsTolerance
	}

	crossProduct := dx*(p.Y-s1.Y) - dy*(p.X-s1.X)
	// crossProduct represents the scaled perpendicular distance.
	// We normalize it by the segment length squared to check the actual distance.
	return (crossProduct*crossProduct)/lengthSq < segmentsTolerance*segmentsTolerance
}

// isPointOnSegment determines if a point p lies on the line segment defined by endpoints s1 and s2 within a tolerance.
func isPointOnSegment(s1, s2, p XY) bool {
	dx := s2.X - s1.X
	dy := s2.Y - s1.Y

	// Squared length of segment B
	squaredLength := dx*dx + dy*dy

	// Prevent division by zero (S1 and S2 coincide)
	if squaredLength < segmentsTolerance {
		distSq := (p.X-s1.X)*(p.X-s1.X) + (p.Y-s1.Y)*(p.Y-s1.Y)
		return distSq < segmentsTolerance
	}

	// Calculate T along the parametric equation of the line: P = S1 + t*(S2 - S1)
	t := ((p.X-s1.X)*dx + (p.Y-s1.Y)*dy) / squaredLength

	// If T is between 0 and 1, the point is within the segment boundaries.
	// We use tolerance to forgive slight imprecision at the extreme vertices.
	return t >= -segmentsTolerance && t <= 1.0+segmentsTolerance
}
