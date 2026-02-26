package wad

import (
	"math"

	"github.com/markel1974/godoom/engine/model"
)

// ScaleFactor is a constant used to scale coordinates or dimensions when transforming data within the engine.
const ScaleFactor = 5.0

//const subSectorBit = uint16(0x8000)

// Polygon represents a collection of points in 2D space that define the vertices of a polygon.
type Polygon []model.XY

// EuclideanDistance calculates the straight-line distance between two points in 2D space.
func EuclideanDistance(p1, p2 model.XY) float64 {
	return math.Hypot(p2.X-p1.X, p2.Y-p1.Y)
}

// PolygonSplit splits a polygon into front and back parts using a plane defined by a point and a normal vector.
func PolygonSplit(poly []model.XY, nx, ny, ndx, ndy float64) (front, back []model.XY) {
	if len(poly) < 3 {
		return nil, nil
	}

	isFront := make([]bool, len(poly))
	for i, p := range poly {
		// In Doom il lato "front" della partizione è definito da val <= 0
		val := ndx*(p.Y-ny) - ndy*(p.X-nx)
		isFront[i] = val <= 1e-5
	}

	for i := 0; i < len(poly); i++ {
		p1 := poly[i]
		p2 := poly[(i+1)%len(poly)]
		f1 := isFront[i]
		f2 := isFront[(i+1)%len(poly)]

		if f1 {
			front = append(front, p1)
		} else {
			back = append(back, p1)
		}

		// Generazione del vertice sul taglio (intersezione)
		if f1 != f2 {
			dx, dy := p2.X-p1.X, p2.Y-p1.Y
			den := ndy*dx - ndx*dy
			if math.Abs(den) > 1e-10 {
				u := (ndx*(p1.Y-ny) - ndy*(p1.X-nx)) / den
				inter := model.XY{X: p1.X + u*dx, Y: p1.Y + u*dy}
				front = append(front, inter)
				back = append(back, inter)
			}
		}
	}
	return PolygonClean(front), PolygonClean(back)
}

// PolygonClean removes duplicate or nearly identical consecutive points from a polygon and ensures the result has at least 3 points.
func PolygonClean(poly []model.XY) []model.XY {
	if len(poly) < 3 {
		return nil
	}
	var res []model.XY
	for _, p := range poly {
		if len(res) == 0 || EuclideanDistance(res[len(res)-1], p) > 0.01 {
			res = append(res, p)
		}
	}
	if len(res) > 1 && EuclideanDistance(res[0], res[len(res)-1]) <= 0.01 {
		res = res[:len(res)-1]
	}
	if len(res) < 3 {
		return nil
	}
	return res
}
