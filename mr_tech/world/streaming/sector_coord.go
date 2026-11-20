package streaming

import "math"

// SectorCoord represents the discrete 2D grid coordinates of a world streaming cell.
type SectorCoord struct {
	X int
	Y int
}

// WorldToSector converts continuous world coordinates (x, y) to discrete sector coordinates for a given cell size.
func WorldToSector(x, y, cellSize float64) SectorCoord {
	if cellSize <= 0.0 {
		cellSize = 256.0
	}
	return SectorCoord{
		X: int(math.Floor(x / cellSize)),
		Y: int(math.Floor(y / cellSize)),
	}
}

// SectorToWorldBounds returns the 2D bounding min and max coordinates (minX, minY, maxX, maxY) for this sector.
func (c SectorCoord) SectorToWorldBounds(cellSize float64) (minX, minY, maxX, maxY float64) {
	if cellSize <= 0.0 {
		cellSize = 256.0
	}
	minX = float64(c.X) * cellSize
	minY = float64(c.Y) * cellSize
	maxX = minX + cellSize
	maxY = minY + cellSize
	return minX, minY, maxX, maxY
}

// ChebyshevDistance returns the maximum coordinate distance (grid ring radius) between c and other.
func (c SectorCoord) ChebyshevDistance(other SectorCoord) int {
	dx := c.X - other.X
	if dx < 0 {
		dx = -dx
	}
	dy := c.Y - other.Y
	if dy < 0 {
		dy = -dy
	}
	if dx > dy {
		return dx
	}
	return dy
}

// ManhattanDistance returns the grid Manhattan distance |dx| + |dy| between c and other.
func (c SectorCoord) ManhattanDistance(other SectorCoord) int {
	dx := c.X - other.X
	if dx < 0 {
		dx = -dx
	}
	dy := c.Y - other.Y
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}

// DistanceSq returns the squared Euclidean grid distance between c and other.
func (c SectorCoord) DistanceSq(other SectorCoord) int {
	dx := c.X - other.X
	dy := c.Y - other.Y
	return dx*dx + dy*dy
}
