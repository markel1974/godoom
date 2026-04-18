package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/model/mathematic"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Face represents a boundary or edge of a Sector, defined by its geometry, connectivity, and optional metadata.
type Face struct {
	parent     *Volume
	neighbor   *Volume
	gScale     float64
	tag        string
	aabb       *physics.AABB
	tri        [3]geometry.XYZ
	normal     geometry.XYZ
	animations []*textures.Animation
	material   *textures.Animation
	minZ       float64
	maxZ       float64
	hasFixedZ  bool
	u          [3]float64
	v          [3]float64
	lockUV     bool
}

// NewFace2d creates a new Face with specified geometry, type, associated neighbor, tag, and texture animations.
func NewFace2d(gScale float64, neighbor *Volume, start geometry.XY, end geometry.XY, tag string, animations []*textures.Animation) *Face {
	out := &Face{
		gScale:     gScale,
		hasFixedZ:  true,
		neighbor:   neighbor,
		tag:        tag,
		minZ:       0,
		maxZ:       0,
		aabb:       physics.NewAABB(),
		animations: []*textures.Animation{nil},
		lockUV:     false,
	}
	if len(animations) > 0 {
		out.animations = animations
	}
	out.tri[0] = geometry.XYZ{X: start.X, Y: start.Y, Z: 0}
	out.tri[1] = geometry.XYZ{X: (start.X + end.X) * 0.5, Y: (start.Y + end.Y) * 0.5, Z: 0}
	out.tri[2] = geometry.XYZ{X: end.X, Y: end.Y, Z: 0}
	out.Rebuild()
	return out
}

// NewFace creates a new 3D segment with specified neighbor, kind, points, tag, and material, and computes its normal and AABB.
func NewFace(gScale float64, neighbor *Volume, tri [3]geometry.XYZ, tag string, material *textures.Animation) *Face {
	out := &Face{
		gScale:    gScale,
		hasFixedZ: false,
		neighbor:  neighbor,
		tag:       tag,
		material:  material,
		aabb:      physics.NewAABB(),
		tri:       tri,
	}
	out.Rebuild()
	return out
}

// LockUV locks or unlocks the UV coordinates of a Face based on the provided staticUV parameter.
func (s *Face) LockUV(lockUV bool) {
	s.lockUV = lockUV
}

// SetUV sets the UV texture coordinates for the three vertices of the face.
func (s *Face) SetUV(u0, v0, u1, v1, u2, v2 float64) {
	s.u[0], s.v[0] = u0, v0
	s.u[1], s.v[1] = u1, v1
	s.u[2], s.v[2] = u2, v2
}

// GetUV retrieves the `u` and `v` coordinate arrays representing the UV mapping of the face.
func (s *Face) GetUV() ([3]float64, [3]float64) {
	return s.u, s.v
}

// SetZ sets the minimum and maximum Z coordinates for the location, marks it as having custom Z bounds, and rebuilds its AABB.
func (s *Face) SetZ(minZ, maxZ float64) {
	s.minZ = minZ
	s.maxZ = maxZ
	s.hasFixedZ = true
	s.tri[0].Z = minZ
	s.tri[1].Z = minZ
	s.tri[2].Z = minZ
	s.Rebuild()
}

// ClearZ resets the Z-coordinate bounds of the location, marks it as lacking custom Z bounds, and triggers a rebuild.
func (s *Face) ClearZ() {
	if s.hasFixedZ {
		s.tri[0].Z = 0
		s.tri[1].Z = 0
		s.tri[2].Z = 0
	}
	s.minZ = 0
	s.maxZ = 0
	s.hasFixedZ = false
	s.Rebuild()
}

// GetParent retrieves the parent Sector of the Face instance. Returns nil if no parent is set.
func (s *Face) GetParent() *Volume {
	return s.parent
}

// SetParent assigns a parent sector to the segment.
func (s *Face) SetParent(parent *Volume) {
	s.parent = parent
}

// GetNeighbor returns the neighboring Sector associated with the Face.
func (s *Face) GetNeighbor() *Volume {
	return s.neighbor
}

// SetNeighbor sets the neighbor sector of the segment. It establishes a link to an adjacent sector.
func (s *Face) SetNeighbor(neighbor *Volume) {
	s.neighbor = neighbor
}

// GetTag returns the tag associated with the segment.
func (s *Face) GetTag() string {
	return s.tag
}

// SetTag sets the tag field of the Face to the specified string value.
func (s *Face) SetTag(tag string) {
	s.tag = tag
}

// GetNormal returns the precomputed normal vector (geometry.XYZ) of the Face.
func (s *Face) GetNormal() geometry.XYZ {
	return s.normal
}

// GetStart returns the first point of the segment as a geometry.XYZ value.
func (s *Face) GetStart() geometry.XYZ {
	return s.tri[0]
}

// GetMiddle retrieves the middle point (geometry.XYZ) of the Face from its predefined points array.
func (s *Face) GetMiddle() geometry.XYZ {
	return s.tri[1]
}

// GetEnd returns the last point of the segment as a geometry.XYZ value.
func (s *Face) GetEnd() geometry.XYZ {
	return s.tri[2]
}

// GetAnimationIndex retrieves the Animation object corresponding to the given material index.
func (s *Face) GetAnimationIndex(m int) *textures.Animation {
	//0 Upper, 1 Middle, 2 Lower
	idx := m % len(s.animations)
	return s.animations[idx]
}

// GetMaterialDetails retrieves the material's texture, type, width scale, and height scale for the face.
func (s *Face) GetMaterialDetails() (*textures.Texture, int) {
	if s.material == nil {
		return nil, 0
	}
	return s.material.CurrentFrame(), s.material.Kind()
}

// GetMaterial returns the root texture material of the face, or nil if it does not exist.
func (s *Face) GetMaterial() *textures.Texture {
	if s.material == nil {
		return nil
	}
	return s.material.CurrentFrame()
}

// GetPoints returns the list of 3D points (geometry.XYZ) that define the segment's shape or path.
func (s *Face) GetPoints() [3]geometry.XYZ {
	return s.tri
}

// PointInLineSide determines if a 2D point (px, py) lies on or to the right of the directed line segment of the Face.
func (s *Face) PointInLineSide(px, py float64) bool {
	start := s.GetStart()
	end := s.GetEnd()
	dir := mathematic.PointInLineDirectionF(px, py, start.X, start.Y, end.X, end.Y)
	if dir < 0 {
		return false
	}
	return true
}

// RayIntersect lancia un raggio. Per evitare singolarità nei BSP,
// chiamala con direzioni irrazionali, es: s.RayIntersect(px, py, pz, 1.0, 0.000123, 0.000456)
func (s *Face) RayIntersect(px, py, pz, dx, dy, dz float64) bool {
	const eps = 1e-8
	p0, p1, p2 := s.tri[0], s.tri[1], s.tri[2]

	e1x, e1y, e1z := p1.X-p0.X, p1.Y-p0.Y, p1.Z-p0.Z
	e2x, e2y, e2z := p2.X-p0.X, p2.Y-p0.Y, p2.Z-p0.Z

	hx := dy*e2z - dz*e2y
	hy := dz*e2x - dx*e2z
	hz := dx*e2y - dy*e2x

	a := e1x*hx + e1y*hy + e1z*hz
	if a > -eps && a < eps {
		return false
	}

	invDet := 1.0 / a
	sx, sy, sz := px-p0.X, py-p0.Y, pz-p0.Z

	u := (sx*hx + sy*hy + sz*hz) * invDet
	if u < 0.0 || u > 1.0 {
		return false
	}

	qx := sy*e1z - sz*e1y
	qy := sz*e1x - sx*e1z
	qz := sx*e1y - sy*e1x

	v := (dx*qx + dy*qy + dz*qz) * invDet
	if v < 0.0 || u+v > 1.0 {
		return false
	}

	t := (e2x*qx + e2y*qy + e2z*qz) * invDet
	return t > eps
}

// PointInside2d determines if the provided 2D point (px, py) lies inside the tri defined by the Face's first three points.
func (s *Face) PointInside2d(px, py float64) bool {
	p0, p1, p2 := s.tri[0], s.tri[1], s.tri[2]
	d1 := (px-p0.X)*(p1.Y-p0.Y) - (py-p0.Y)*(p1.X-p0.X)
	d2 := (px-p1.X)*(p2.Y-p1.Y) - (py-p1.Y)*(p2.X-p1.X)
	d3 := (px-p2.X)*(p0.Y-p2.Y) - (py-p2.Y)*(p0.X-p2.X)
	const eps = -0.001
	hasNeg := (d1 < eps) || (d2 < eps) || (d3 < eps)
	hasPos := (d1 > -eps) || (d2 > -eps) || (d3 > -eps)
	return !(hasNeg && hasPos)
}

// PointInside3d determina se il punto 3D (px, py, pz) giace all'interno del triangolo.
// Utilizza il calcolo delle Coordinate Baricentriche per la massima efficienza.
func (s *Face) PointInside3d(px, py, pz float64) bool {
	p0, p1, p2 := s.tri[0], s.tri[1], s.tri[2]
	// 1. Calcolo dei vettori degli spigoli (v0, v1) e del vettore verso il punto (v2)
	v0x, v0y, v0z := p2.X-p0.X, p2.Y-p0.Y, p2.Z-p0.Z
	v1x, v1y, v1z := p1.X-p0.X, p1.Y-p0.Y, p1.Z-p0.Z
	v2x, v2y, v2z := px-p0.X, py-p0.Y, pz-p0.Z
	// 2. Calcolo dei Prodotti Scalari (Dot Products)
	d00 := v0x*v0x + v0y*v0y + v0z*v0z
	d01 := v0x*v1x + v0y*v1y + v0z*v1z
	d02 := v0x*v2x + v0y*v2y + v0z*v2z
	d11 := v1x*v1x + v1y*v1y + v1z*v1z
	d12 := v1x*v2x + v1y*v2y + v1z*v2z
	// 3. Calcolo del denominatore
	denom := (d00 * d11) - (d01 * d01)
	if denom == 0 {
		return false // Sicurezza: Triangolo degenere (linea o punto)
	}
	invDenom := 1.0 / denom
	// 4. Calcolo delle coordinate baricentriche (u, v)
	u := ((d11 * d02) - (d01 * d12)) * invDenom
	v := ((d00 * d12) - (d01 * d02)) * invDenom
	// 5. Verifica tolleranza (eps) per la virgola mobile
	const eps = -0.001
	// Il punto è DENTRO il triangolo se u >= 0, v >= 0 e u+v <= 1
	// (usiamo la tua tolleranza eps per prevenire errori di arrotondamento sui bordi)
	return (u >= eps) && (v >= eps) && (u+v <= 1.0-eps)
}

// Scale2d scales the starting and ending points of the segment by applying the given scale factor.
func (s *Face) Scale2d(scale float64) {
	s.tri[0].Scale(scale)
	s.tri[1].Scale(scale)
	s.tri[2].Scale(scale)
	s.Rebuild()
}

// Rebuild calculates the axis-aligned bounding box (AABB) for the segment, considering both 2D and 3D cases.
func (s *Face) Rebuild() {
	s.computeAABB()
	normal := s.computeNormal()
	s.computeUV(normal)
}

// GetAABB returns the axis-aligned bounding box (AABB) associated with the segment.
func (s *Face) GetAABB() *physics.AABB {
	return s.aabb
}

// computeNormal calculates and assigns the normal vector (geometry.XYZ) for the Face based on its points and geometry.
func (s *Face) computeNormal() geometry.XYZ {
	s.normal = geometry.XYZ{X: 0, Y: 0, Z: 1}
	if s.hasFixedZ {
		p0, p1 := s.tri[0], s.tri[2]
		dx := p1.X - p0.X
		dy := p1.Y - p0.Y
		lenSq := dx*dx + dy*dy
		if lenSq > 0 {
			invLen := 1.0 / math.Sqrt(lenSq)
			// Proiezione del vettore normale 2D nello spazio 3D
			s.normal = geometry.XYZ{X: -dy * invLen, Y: dx * invLen, Z: 0}
		}
		return s.normal
	}
	// Prodotto vettoriale standard per poligoni 3D
	p0, p1, p2 := s.tri[0], s.tri[1], s.tri[2]
	v1x, v1y, v1z := p1.X-p0.X, p1.Y-p0.Y, p1.Z-p0.Z
	v2x, v2y, v2z := p2.X-p0.X, p2.Y-p0.Y, p2.Z-p0.Z
	nx := v1y*v2z - v1z*v2y
	ny := v1z*v2x - v1x*v2z
	nz := v1x*v2y - v1y*v2x
	l := math.Sqrt(nx*nx + ny*ny + nz*nz)
	if l > 0 {
		s.normal = geometry.XYZ{X: nx / l, Y: ny / l, Z: nz / l}
	}
	return s.normal
}

// computeAABB calculates the axis-aligned bounding box (AABB) for the Face using its points and optional Z bounds.
func (s *Face) computeAABB() {
	const eps = 0.001
	minX, minY, minZ := s.tri[0].X, s.tri[0].Y, s.tri[0].Z
	maxX, maxY, maxZ := minX, minY, minZ
	for _, p := range s.tri {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
		if p.Z < minZ {
			minZ = p.Z
		}
		if p.Z > maxZ {
			maxZ = p.Z
		}
	}
	if s.hasFixedZ {
		minZ = s.minZ
		maxZ = s.maxZ
	} else {
		if minZ == maxZ {
			minZ -= eps
			maxZ += eps
		}
	}
	s.aabb.Rebuild(minX-eps, minY-eps, minZ, maxX+eps, maxY+eps, maxZ)
}

// computeUV computes the UV mapping for the current face based on its normal, material, and texture scaling factors.
func (s *Face) computeUV(normal geometry.XYZ) {
	if s.lockUV {
		return
	}
	if s.material == nil {
		return
	}
	tex := s.material.CurrentFrame()
	if tex == nil {
		return
	}
	texW, texH := tex.Size()
	texScaleW, texScaleH := s.material.ScaleFactor()
	if texScaleW == 0 {
		texScaleW = 1.0
	}
	if texScaleH == 0 {
		texScaleH = 1.0
	}
	gScale := s.gScale
	baseW := float64(texW) / gScale
	baseH := float64(texH) / gScale
	w := baseW * texScaleW
	h := baseH * texScaleH
	absX := math.Abs(normal.X)
	absY := math.Abs(normal.Y)
	absZ := math.Abs(normal.Z)

	//todo invertire U/V nel modello, poi il renderer utilizzera il metodo migliore per visualizzare

	/*
			// Proiezione Triplanare Pura.
			    // Nessuna inversione, nessuna compensazione OpenGL.
			    if absZ >= absX && absZ >= absY {
			       // Upper / Lower (Pavimenti e Soffitti)
			       s.SetUV(s.tri[0].X/w, s.tri[0].Y/h, s.tri[1].X/w, s.tri[1].Y/h, s.tri[2].X/w, s.tri[2].Y/h)
			    } else if absY >= absX && absY >= absZ {
			       // Muri rivolti su Y
			       s.SetUV(s.tri[0].X/w, s.tri[0].Z/h, s.tri[1].X/w, s.tri[1].Z/h, s.tri[2].X/w, s.tri[2].Z/h)
			    } else {
			       // Muri rivolti su X
			       s.SetUV(s.tri[0].Y/w, s.tri[0].Z/h, s.tri[1].Y/w, s.tri[1].Z/h, s.tri[2].Y/w, s.tri[2].Z/h)
			    }
		NEL RENDERER
		id0 := w.fv.AddVertex(float32(p[0].X), float32(p[0].Z), float32(-p[0].Y), float32(u[0]), float32(-v[0]), l, oX, oY, oZ, b)
		id1 := w.fv.AddVertex(float32(p[1].X), float32(p[1].Z), float32(-p[1].Y), float32(u[1]), float32(-v[1]), l, oX, oY, oZ, b)
		id2 := w.fv.AddVertex(float32(p[2].X), float32(p[2].Z), float32(-p[2].Y), float32(u[2]), float32(-v[2]), l, oX, oY, oZ, b)
	*/

	// Proiezione Triplanare Z-UP compensata per OpenGL (X, Z, -Y)
	if absZ >= absX && absZ >= absY {
		// Upper / Lower (Pavimenti e Soffitti)
		// L'asse Y nel mondo logico diventa la profondità (Z) in OpenGL,
		// invertiamo Y per mantenere l'orientamento della texture.
		s.SetUV(s.tri[0].X/w, -s.tri[0].Y/h, s.tri[1].X/w, -s.tri[1].Y/h, s.tri[2].X/w, -s.tri[2].Y/h)
	} else if absY >= absX && absY >= absZ {
		// Muri rivolti su Y
		// L'asse Z è l'altezza. Mettiamo il meno (-Z) in modo che quando Z sale, V scenda.
		s.SetUV(s.tri[0].X/w, -s.tri[0].Z/h, s.tri[1].X/w, -s.tri[1].Z/h, s.tri[2].X/w, -s.tri[2].Z/h)
	} else {
		// Muri rivolti su X
		// L'asse Z è l'altezza. Mettiamo il meno (-Z).
		s.SetUV(s.tri[0].Y/w, -s.tri[0].Z/h, s.tri[1].Y/w, -s.tri[1].Z/h, s.tri[2].Y/w, -s.tri[2].Z/h)
	}
}

// PointInVolume checks if a point (px, py, pz) lies within the Face's location. Returns distance and a boolean status.
//func (s *Face) PointInVolume(px, py, pz float64) (float64, bool) {
//	target := s.tri[0]
//	pointInVolume := (px-target.X)*s.normal.X + (py-target.Y)*s.normal.Y + (pz-target.Z)*s.normal.Z
//	return pointInVolume, true
//}

/*
// RayIntersect determines if a ray starting at the origin (1, 0, 0) intersects with the tri of the face.
// The method uses the Möller-Trumbore intersection algorithm for precise calculations.
// px, py, pz parameters specify the coordinates of the point relative to which the intersection occurs.
// Returns true if the ray intersects the tri and false otherwise.
func (s *Face) RayIntersect(px, py, pz float64) bool {
	const eps = 0.00001
	p0, p1, p2 := s.tri[0], s.tri[1], s.tri[2]
	// 1. Estrai gli edge del triangolo
	e1x, e1y, e1z := p1.X-p0.X, p1.Y-p0.Y, p1.Z-p0.Z
	e2x, e2y, e2z := p2.X-p0.X, p2.Y-p0.Y, p2.Z-p0.Z
	// 2. Cross Product tra Raggio(1,0,0) ed Edge2.
	// Dir x E2 = (0, -E2z, E2y)
	hy, hz := -e2z, e2y
	// 3. Determinante (a = Edge1 dot h)
	a := e1y*hy + e1z*hz
	if math.Abs(a) < eps {
		return false // Il raggio è esattamente parallelo al triangolo (o triangolo degenere)
	}
	invDet := 1.0 / a
	// 4. Distanza del punto P dal vertice 0 (s = P - P0)
	sx, sy, sz := px-p0.X, py-p0.Y, pz-p0.Z
	// 5. Parametro Baricentrico U
	u := invDet * (sy*hy + sz*hz)
	if u < 0.0 || u > 1.0 {
		return false // L'intersezione manca il triangolo su questo asse
	}
	// 6. Cross Product s x Edge1 (q)
	qx := sy*e1z - sz*e1y
	qy := sz*e1x - sx*e1z
	qz := sx*e1y - sy*e1x
	// 7. Parametro Baricentrico V
	// v = invDet * (Dir dot q). Poiché Dir = (1,0,0), il dot è semplicemente qx!
	v := invDet * qx
	if v < 0.0 || u+v > 1.0 {
		return false // L'intersezione manca il triangolo
	}
	// 8. Calcolo del Time Of Impact (t) lungo il raggio
	t := invDet * (e2x*qx + e2y*qy + e2z*qz)
	// Se t > eps, il triangolo è davanti a noi e l'abbiamo colpito
	return t > eps
}*/

// MakeStraightEdgeKey generates an EdgeKey for the segment using its start and end points, based on a fixed precision.
//func (s *Face) MakeStraightEdgeKey() EdgeKey {
//	return makeEdgeKey(edgePrecision, s.GetStart(), s.GetEnd())
//}

// MakeReverseEdgeKey generates an EdgeKey by reversing the start and end points of the segment with a fixed precision.
//func (s *Face) MakeReverseEdgeKey() EdgeKey {
//	return makeEdgeKey(edgePrecision, s.GetEnd(), s.GetStart())
//}

/*
// edgePrecision defines the scaling factor used to convert floating-point coordinates into integer-based EdgeKey components.
const edgePrecision = 1000.0

// EdgeKey represents a unique identifier for an edge in 2D space, defined by the rounded coordinates of its endpoints.
type EdgeKey struct {
	x1, y1, x2, y2 int64
}

// makeEdgeKey generates an EdgeKey by scaling and rounding the start and end coordinates using the given precision.
func makeEdgeKey(precision float64, start geometry.XYZ, end geometry.XYZ) EdgeKey {
	return EdgeKey{
		x1: int64(math.Round(start.X * precision)),
		y1: int64(math.Round(start.Y * precision)),
		x2: int64(math.Round(end.X * precision)),
		y2: int64(math.Round(end.Y * precision)),
	}
}*/
