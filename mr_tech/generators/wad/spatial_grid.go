package wad

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/generators/wad/geometry"
	"github.com/markel1974/godoom/mr_tech/model"
)

// CellKey represents a grid cell in a 2D spatial grid, defined by integer X and Y coordinates.
type CellKey struct {
	X, Y int
}

// SpatialGrid represents a spatial partitioning system for efficient management of ConfigSectors in a 2D grid.
type SpatialGrid struct {
	cellSize float64
	cells    map[CellKey][]*model.ConfigSector
	all      []*model.ConfigSector // Retained for global fallback
}

// NewSpatialGrid constructs a SpatialGrid, partitioning ConfigSector objects into cells based on the given cell size.
func NewSpatialGrid(sectors []*model.ConfigSector, cellSize float64) *SpatialGrid {
	grid := &SpatialGrid{
		cellSize: cellSize,
		cells:    make(map[CellKey][]*model.ConfigSector),
		all:      sectors,
	}

	for _, s := range sectors {
		if len(s.Segments) != 3 {
			continue
		}

		// Note: Y coordinates are inverted as in the original code
		v1 := geometry.Point{X: s.Segments[0].Start.X, Y: -s.Segments[0].Start.Y}
		v2 := geometry.Point{X: s.Segments[1].Start.X, Y: -s.Segments[1].Start.Y}
		v3 := geometry.Point{X: s.Segments[2].Start.X, Y: -s.Segments[2].Start.Y}

		minX := math.Min(v1.X, math.Min(v2.X, v3.X))
		maxX := math.Max(v1.X, math.Max(v2.X, v3.X))
		minY := math.Min(v1.Y, math.Min(v2.Y, v3.Y))
		maxY := math.Max(v1.Y, math.Max(v2.Y, v3.Y))

		startX := int(math.Floor(minX / cellSize))
		endX := int(math.Floor(maxX / cellSize))
		startY := int(math.Floor(minY / cellSize))
		endY := int(math.Floor(maxY / cellSize))

		for x := startX; x <= endX; x++ {
			for y := startY; y <= endY; y++ {
				k := CellKey{x, y}
				grid.cells[k] = append(grid.cells[k], s)
			}
		}
	}
	return grid
}

// ResolveSectorId determines the sector ID for a given point within the spatial grid, considering grid cells and fallbacks.
func (grid *SpatialGrid) ResolveSectorId(p geometry.Point) string {
	if len(grid.all) == 0 {
		return ""
	}

	k := CellKey{int(math.Floor(p.X / grid.cellSize)), int(math.Floor(p.Y / grid.cellSize))}
	candidates := grid.cells[k]

	// Fast path: Point falls inside a populated grid cell
	if len(candidates) > 0 {
		var minDist = math.MaxFloat64
		closestSector := candidates[0].Id

		for _, s := range candidates {
			v1 := geometry.Point{X: s.Segments[0].Start.X, Y: -s.Segments[0].Start.Y}
			v2 := geometry.Point{X: s.Segments[1].Start.X, Y: -s.Segments[1].Start.Y}
			v3 := geometry.Point{X: s.Segments[2].Start.X, Y: -s.Segments[2].Start.Y}

			if grid.PointInTriangle(p, v1, v2, v3) {
				return s.Id
			}

			cx := (v1.X + v2.X + v3.X) / 3.0
			cy := (v1.Y + v2.Y + v3.Y) / 3.0
			distSq := (cx-p.X)*(cx-p.X) + (cy-p.Y)*(cy-p.Y)

			if distSq < minDist {
				minDist = distSq
				closestSector = s.Id
			}
		}
		return closestSector
	}

	// Slow path: Point is in the void. Fallback to global nearest centroid.
	var minDist = math.MaxFloat64
	closestSector := grid.all[0].Id

	for _, s := range grid.all {
		if len(s.Segments) != 3 {
			continue
		}
		v1 := geometry.Point{X: s.Segments[0].Start.X, Y: -s.Segments[0].Start.Y}
		v2 := geometry.Point{X: s.Segments[1].Start.X, Y: -s.Segments[1].Start.Y}
		v3 := geometry.Point{X: s.Segments[2].Start.X, Y: -s.Segments[2].Start.Y}

		cx := (v1.X + v2.X + v3.X) / 3.0
		cy := (v1.Y + v2.Y + v3.Y) / 3.0
		distSq := (cx-p.X)*(cx-p.X) + (cy-p.Y)*(cy-p.Y)

		if distSq < minDist {
			minDist = distSq
			closestSector = s.Id
		}
	}
	return closestSector
}

// PointInTriangle determines if a point lies inside or on the edges of a triangle defined by three vertices.
// Returns true if the point is inside the triangle, false otherwise. Uses vector cross-products for calculations.
func (grid *SpatialGrid) PointInTriangle(p geometry.Point, a geometry.Point, b geometry.Point, c geometry.Point) bool {
	cp1 := (b.X-a.X)*(p.Y-a.Y) - (b.Y-a.Y)*(p.X-a.X)
	cp2 := (c.X-b.X)*(p.Y-b.Y) - (c.Y-b.Y)*(p.X-b.X)
	cp3 := (a.X-c.X)*(p.Y-c.Y) - (a.Y-c.Y)*(p.X-c.X)

	const eps = 0.5
	return (cp1 >= -eps && cp2 >= -eps && cp3 >= -eps) || (cp1 <= eps && cp2 <= eps && cp3 <= eps)
}
