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
	tag        string
	aabb       *physics.AABB
	tri        [3]geometry.XYZ
	normal     geometry.XYZ
	normalAbs  geometry.XYZ
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
func NewFace2d(neighbor *Volume, start geometry.XY, end geometry.XY, tag string, animations []*textures.Animation) *Face {
	out := &Face{
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

// NewFace creates a new 3D segment with specified neighbor, stage, points, tag, and material, and computes its normal and AABB.
func NewFace(neighbor *Volume, tri [3]geometry.XYZ, tag string, material *textures.Animation) *Face {
	out := &Face{
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
	s.computeNormal()
	s.computeUV()
}

// GetAABB returns the axis-aligned bounding box (AABB) associated with the segment.
func (s *Face) GetAABB() *physics.AABB {
	return s.aabb
}

func (s *Face) SweepTest(viewX, viewY, viewZ, px, py, pz, velX, velY, velZ, radius float64) (float64, float64, float64, float64, bool) {
	n := s.normal
	p0 := s.tri[0]
	p1 := s.tri[1]
	p2 := s.tri[2]

	vDotN := velX*n.X + velY*n.Y + velZ*n.Z
	distStart := (viewX-p0.X)*n.X + (viewY-p0.Y)*n.Y + (viewZ-p0.Z)*n.Z

	// ==========================================
	// FIX DEFINITIVO: DOUBLE-SIDED DINAMICO
	// ==========================================
	invertNormal := false

	// Se la velocità va nella stessa direzione della normale, colpiamo da dietro (Faccia Interna)
	if vDotN > 1e-6 {
		invertNormal = true
	} else if math.Abs(vDotN) <= 1e-6 && distStart < 0.0 {
		// Scivolamento parallelo dal lato posteriore
		invertNormal = true
	}

	nx, ny, nz := n.X, n.Y, n.Z

	if invertNormal {
		// Capovolge il mondo localmente per far rimbalzare l'entità anche dai muri "al contrario"
		nx, ny, nz = -nx, -ny, -nz
		distStart = -distStart
		vDotN = -vDotN
		p1, p2 = p2, p1 // Winding flip
	}

	// Backface Culling (Ora è relativo al lato corretto, non scarterà mai un pavimento!)
	if distStart < -radius {
		return 0, 0, 0, 0, false
	}

	var minT = 1.0
	var hit = false
	var cNx, cNy, cNz float64
	const epsilon = 1e-4 // Time-Of-Impact Back-off per scivolare sui bordi senza incastrarsi

	// ==========================================
	// FASE A: IMPATTO SUL PIANO
	// ==========================================
	var t0, t1 float64
	isParallel := false

	if math.Abs(vDotN) < 1e-6 {
		if distStart >= -radius && distStart <= radius {
			t0, t1 = 0.0, 1.0
			isParallel = true
		} else {
			t0, t1 = 2.0, 2.0
		}
	} else {
		t0 = (radius - distStart) / vDotN
		t1 = (-radius - distStart) / vDotN
		if t0 > t1 {
			t0, t1 = t1, t0
		}
	}

	if (vDotN < 0.0 || isParallel) && t0 <= 1.0 && t1 >= 0.0 {
		tPlane := math.Max(0.0, t0) // Micro-penetration clamp

		if tPlane <= 1.0 {
			cx := viewX + velX*tPlane
			cy := viewY + velY*tPlane
			cz := viewZ + velZ*tPlane

			hX := cx - nx*radius
			hY := cy - ny*radius
			hZ := cz - nz*radius

			e1x, e1y, e1z := p1.X-p0.X, p1.Y-p0.Y, p1.Z-p0.Z
			e2x, e2y, e2z := p2.X-p0.X, p2.Y-p0.Y, p2.Z-p0.Z
			vX, vY, vZ := hX-p0.X, hY-p0.Y, hZ-p0.Z

			d00 := e1x*e1x + e1y*e1y + e1z*e1z
			d01 := e1x*e2x + e1y*e2y + e1z*e2z
			d11 := e2x*e2x + e2y*e2y + e2z*e2z
			d20 := vX*e1x + vY*e1y + vZ*e1z
			d21 := vX*e2x + vY*e2y + vZ*e2z

			denom := d00*d11 - d01*d01

			if math.Abs(denom) > 1e-8 {
				v := (d11*d20 - d01*d21) / denom
				w := (d00*d21 - d01*d20) / denom
				u := 1.0 - v - w

				// Saldatura Baricentrica per tappare i micro-buchi della griglia 3D
				const baryEps = -1e-4
				if v >= baryEps && w >= baryEps && u >= baryEps {
					safeT := math.Max(0.0, tPlane-epsilon)
					return safeT, nx, ny, nz, true // Ritorno rapido
				}
			}
		}
	}

	// ==========================================
	// FASE B: SWEEP CONTINUO SU VERTICI E SPIGOLI
	// ==========================================
	velSq := velX*velX + velY*velY + velZ*velZ
	if velSq < 1e-8 {
		return 0, 0, 0, 0, false
	}

	solveQuad := func(a, b, c float64) (float64, bool) {
		// Anti-Tunneling: Se la sfera tocca GIÀ il bordo, fermala istantaneamente.
		if c <= 0.0 {
			return 0.0, true
		}
		det := b*b - 4.0*a*c
		if det < 0.0 {
			return 1.0, false
		}
		sqD := math.Sqrt(det)
		r1 := (-b - sqD) / (2.0 * a)
		r2 := (-b + sqD) / (2.0 * a)

		if r1 > r2 {
			r1, r2 = r2, r1
		}
		if r1 >= 0.0 && r1 <= 1.0 && r1 < minT {
			return r1, true
		}
		if r2 >= 0.0 && r2 <= 1.0 && r2 < minT {
			return r2, true
		}
		return 1.0, false
	}

	// Test Vertici
	pts := [3]geometry.XYZ{p0, p1, p2}
	for _, p := range pts {
		vx, vy, vz := viewX-p.X, viewY-p.Y, viewZ-p.Z
		a := velSq
		b := 2.0 * (velX*vx + velY*vy + velZ*vz)
		c := (vx*vx + vy*vy + vz*vz) - radius*radius

		if t, ok := solveQuad(a, b, c); ok {
			minT = t
			hit = true
			cNx, cNy, cNz = viewX+velX*t-p.X, viewY+velY*t-p.Y, viewZ+velZ*t-p.Z
		}
	}

	// Test Spigoli
	for i := 0; i < 3; i++ {
		pA := pts[i]
		pB := pts[(i+1)%3]

		edgeX, edgeY, edgeZ := pB.X-pA.X, pB.Y-pA.Y, pB.Z-pA.Z
		edgeLenSq := edgeX*edgeX + edgeY*edgeY + edgeZ*edgeZ

		if edgeLenSq < 1e-8 {
			continue
		}

		vx, vy, vz := viewX-pA.X, viewY-pA.Y, viewZ-pA.Z

		edgeDotVel := velX*edgeX + velY*edgeY + velZ*edgeZ
		edgeDotOrig := vx*edgeX + vy*edgeY + vz*edgeZ

		a := edgeLenSq*velSq - edgeDotVel*edgeDotVel
		b := edgeLenSq*2.0*(velX*vx+velY*vy+velZ*vz) - 2.0*edgeDotVel*edgeDotOrig
		c := edgeLenSq*((vx*vx+vy*vy+vz*vz)-radius*radius) - edgeDotOrig*edgeDotOrig

		if a == 0.0 {
			continue
		}

		if t, ok := solveQuad(a, b, c); ok {
			f := (edgeDotOrig + edgeDotVel*t) / edgeLenSq
			if f >= 0.0 && f <= 1.0 {
				minT = t
				hit = true
				closestX := pA.X + f*edgeX
				closestY := pA.Y + f*edgeY
				closestZ := pA.Z + f*edgeZ
				cNx, cNy, cNz = viewX+velX*t-closestX, viewY+velY*t-closestY, viewZ+velZ*t-closestZ
			}
		}
	}

	if hit {
		safeT := math.Max(0.0, minT-epsilon)
		l := math.Sqrt(cNx*cNx + cNy*cNy + cNz*cNz)
		if l > 1e-8 {
			return safeT, cNx / l, cNy / l, cNz / l, true
		}
		// Fallback se la normale dello spigolo degenera
		return safeT, nx, ny, nz, true
	}

	return 0, 0, 0, 0, false
}

// computeNormal calculates and assigns the normal vector (geometry.XYZ) for the Face based on its points and geometry.
func (s *Face) computeNormal() {
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
	} else {
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
	}
	s.normalAbs = geometry.XYZ{
		X: math.Abs(s.normal.X), Y: math.Abs(s.normal.Y), Z: math.Abs(s.normal.Z),
	}
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
func (s *Face) computeUV() {
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
	w, h := tex.GetSizeScaled()
	absX := s.normalAbs.X
	absY := s.normalAbs.Y
	absZ := s.normalAbs.Z
	// Pure Triplanar Projection.
	if absZ >= absX && absZ >= absY {
		// Upper / Lower (Floors and Ceilings)
		u0, v0 := s.tri[0].X/w, s.tri[0].Y/h
		u1, v1 := s.tri[1].X/w, s.tri[1].Y/h
		u2, v2 := s.tri[2].X/w, s.tri[2].Y/h
		s.SetUV(u0, v0, u1, v1, u2, v2)
	} else if absY >= absX && absY >= absZ {
		// Walls facing Y
		s.SetUV(s.tri[0].X/w, s.tri[0].Z/h, s.tri[1].X/w, s.tri[1].Z/h, s.tri[2].X/w, s.tri[2].Z/h)
	} else {
		// Walls facing X
		s.SetUV(s.tri[0].Y/w, s.tri[0].Z/h, s.tri[1].Y/w, s.tri[1].Z/h, s.tri[2].Y/w, s.tri[2].Z/h)
	}
}

/*
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
*/

/*
// RayIntersectDist calculates if a ray intersects the triangle and returns a boolean and the distance to the intersection.
func (s *Face) RayIntersectDist(px, py, pz, dx, dy, dz float64) (bool, float64) {
	const eps = 1e-8
	p0, p1, p2 := s.tri[0], s.tri[1], s.tri[2]

	e1x, e1y, e1z := p1.X-p0.X, p1.Y-p0.Y, p1.Z-p0.Z
	e2x, e2y, e2z := p2.X-p0.X, p2.Y-p0.Y, p2.Z-p0.Z

	hx := dy*e2z - dz*e2y
	hy := dz*e2x - dx*e2z
	hz := dx*e2y - dy*e2x

	a := e1x*hx + e1y*hy + e1z*hz
	// Culling dei raggi paralleli al triangolo
	if a > -eps && a < eps {
		return false, 0.0
	}

	invDet := 1.0 / a
	sx, sy, sz := px-p0.X, py-p0.Y, pz-p0.Z

	u := (sx*hx + sy*hy + sz*hz) * invDet
	if u < 0.0 || u > 1.0 {
		return false, 0.0
	}

	qx := sy*e1z - sz*e1y
	qy := sz*e1x - sx*e1z
	qz := sx*e1y - sy*e1x

	v := (dx*qx + dy*qy + dz*qz) * invDet
	if v < 0.0 || u+v > 1.0 {
		return false, 0.0
	}
	// t è la distanza dall'origine del raggio al punto di intersezione
	t := (e2x*qx + e2y*qy + e2z*qz) * invDet
	// Ritorna true e la distanza solo se l'impatto è in avanti (davanti al raggio)
	if t > eps {
		return true, t
	}
	return false, 0.0
}
*/
