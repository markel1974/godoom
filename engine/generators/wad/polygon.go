package wad

import (
	"math"

	"github.com/markel1974/godoom/engine/model"
)

//const subSectorBit = uint16(0x8000)

// Polygons represents a shape as a slice of 2D points in counter-clockwise order.
type Polygons []model.XY

// EuclideanDistance calculates the Euclidean distance between two points in a 2D space represented as model.XY structs.
func EuclideanDistance(p1 model.XY, p2 model.XY) float64 {
	return math.Hypot(p2.X-p1.X, p2.Y-p1.Y)
}

// SnapFloat rounds a float64 value to 4 decimal places to stabilize floating-point operations and reduce precision errors.
func SnapFloat(val float64) float64 {
	return math.Round(val*10000.0) / 10000.0
}

// PolygonSplit separates a polygon into two parts (front and back) using a partition line defined by its parameters.
// poly is the input polygon, nx and ny represent the line origin, and ndx and ndy define the line direction.
// The function returns two slices: the points on the front side and the points on the back side of the partition.
func PolygonSplit(poly Polygons, nx int16, ny int16, ndx int16, ndy int16) (Polygons, Polygons) {
	var front Polygons
	var back Polygons

	if len(poly) < 3 {
		return nil, nil
	}

	fnx, fny := float64(nx), float64(ny)
	fndx, fndy := float64(ndx), float64(ndy)

	isFront := make([]bool, len(poly))
	for i, p := range poly {
		// In Doom il lato "front" della partizione è definito da val <= 0
		val := fndx*(p.Y-fny) - fndy*(p.X-fnx)
		// Margine per la stabilità in virgola mobile sulle coordinate native
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

		if f1 != f2 {
			dx, dy := p2.X-p1.X, p2.Y-p1.Y
			den := fndy*dx - fndx*dy
			if math.Abs(den) > 1e-10 {
				u := (fndx*(p1.Y-fny) - fndy*(p1.X-fnx)) / den
				interX := SnapFloat(p1.X + u*dx)
				interY := SnapFloat(p1.Y + u*dy)
				inter := model.XY{X: interX, Y: interY}
				//inter := model.XY{X: p1.X + u*dx, Y: p1.Y + u*dy}
				front = append(front, inter)
				back = append(back, inter)
			}
		}
	}
	return PolygonClean(front), PolygonClean(back)
}

// PolygonClean removes duplicate consecutive points and collapses nearly identical edges in a polygon represented by XY points.
// If the resulting polygon has fewer than 3 vertices, it returns nil.
func PolygonClean(poly Polygons) Polygons {
	if len(poly) < 3 {
		return nil
	}
	var res Polygons
	for _, p := range poly {
		// La tolleranza 0.01 è perfetta se lavori in scala originale Doom [-32768, 32768]
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
