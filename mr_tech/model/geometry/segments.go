package geometry

import "math"

const segmentsTolerance = 1e-6 // segmentsTolerance defines the precision threshold for geometric calculations, such as collinearity and point-segment checks.

// IsSegmentSubset determines if the segment defined by points a1 and a2 is a subset of the segment defined by points b1 and b2.
func IsSegmentSubset(a1, a2, b1, b2 XY) bool {
	// 1. Condizione di Collinearità:
	// A1 e A2 devono trovarsi sulla linea infinita passante per B1 e B2.
	if !isCollinear(b1, b2, a1) || !isCollinear(b1, b2, a2) {
		return false
	}

	// 2. Condizione di Inclusione:
	// Entrambi i punti A1 e A2 devono trovarsi "tra" B1 e B2.
	return isPointOnSegment(b1, b2, a1) && isPointOnSegment(b1, b2, a2)
}

// isCollinear determines if a point p is collinear with a line segment defined by points s1 and s2, using a tolerance.
func isCollinear(s1, s2, p XY) bool {
	crossProduct := (s2.X-s1.X)*(p.Y-s1.Y) - (s2.Y-s1.Y)*(p.X-s1.X)
	return math.Abs(crossProduct) < segmentsTolerance
}

// isPointOnSegment determines if a point p lies on the line segment defined by endpoints s1 and s2 within a tolerance.
func isPointOnSegment(s1, s2, p XY) bool {
	dx := s2.X - s1.X
	dy := s2.Y - s1.Y

	// Lunghezza al quadrato del segmento B
	squaredLength := dx*dx + dy*dy

	// Prevenzione divisione per zero (S1 e S2 coincidono)
	if squaredLength < segmentsTolerance {
		distSq := (p.X-s1.X)*(p.X-s1.X) + (p.Y-s1.Y)*(p.Y-s1.Y)
		return distSq < segmentsTolerance
	}

	// Calcolo di T lungo l'equazione parametrica della retta: P = S1 + t*(S2 - S1)
	t := ((p.X-s1.X)*dx + (p.Y-s1.Y)*dy) / squaredLength

	// Se T è tra 0 e 1, il punto è dentro i limiti del segmento.
	// Usiamo tolerance per perdonare lievi sbavature sui vertici estremi.
	return t >= -segmentsTolerance && t <= 1.0+segmentsTolerance
}
