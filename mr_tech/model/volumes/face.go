package volumes

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Face represents a boundary or edge of a Sector, defined by its geometry, connectivity, and optional metadata.
type Face struct {
	id       int
	parent   *Volume
	label    string
	aabb     *aabb.AABB
	tri      *geometry.Triangle3D
	n        geometry.XYZ
	nAbs     geometry.XYZ
	material *textures.Material
	u        [3]float64
	v        [3]float64
	lockUV   bool
}

func NewFaceEmpty(material *textures.Material, u, v [3]float64) *Face {
	out := &Face{
		material: material,
		tri:      geometry.NewTriangle3D([3]geometry.XYZ{}),
		u:        u,
		v:        v,
	}
	return out
}

// NewFace creates a new 3D segment with specified neighbor, stage, points, tag, and material, and computes its n and AABB.
func NewFace(tri [3]geometry.XYZ, label string, material *textures.Material) *Face {
	out := &Face{
		label:    label,
		material: material,
		aabb:     aabb.NewAABB(),
		tri:      geometry.NewTriangle3D(tri),
	}
	out.Rebuild()
	return out
}

// GetTri returns the three vertices that define the face triangle.
func (fc *Face) GetTri() *geometry.Triangle3D {
	return fc.tri
}

// Get0 returns the x, y, and z coordinates of the face's first vertex.
func (fc *Face) Get0() (float64, float64, float64) {
	return fc.tri.GetPoint0()
}

// Get1 returns the x, y, and z coordinates of the face's first triangle point.
func (fc *Face) Get1() (float64, float64, float64) {
	return fc.tri.GetPoint1()
}

// Get2 returns the coordinates of the face's second point.
func (fc *Face) Get2() (float64, float64, float64) {
	return fc.tri.GetPoint2()
}

// Translate shifts the coordinates of a triangle vertex by specified offsets in X, Y, and Z directions.
func (fc *Face) Translate(idx int, offX, offY, offZ float64) (float64, float64, float64) {
	return fc.tri.Translate(idx, offX, offY, offZ)
}

// LockUV locks or unlocks the UV coordinates of a Face based on the provided staticUV parameter.
func (fc *Face) LockUV(lockUV bool) {
	fc.lockUV = lockUV
}

// SetUV sets the UV texture coordinates for the three vertices of the face.
func (fc *Face) SetUV(u0, v0, u1, v1, u2, v2 float64) {
	fc.u[0], fc.v[0] = u0, v0
	fc.u[1], fc.v[1] = u1, v1
	fc.u[2], fc.v[2] = u2, v2
}

// GetUV retrieves the `u` and `v` coordinate arrays representing the UV mapping of the face.
func (fc *Face) GetUV() ([3]float64, [3]float64) {
	return fc.u, fc.v
}

// GetUV0 retrieves the UV texture coordinates (u, v) for the first vertex of the triangular face.
func (fc *Face) GetUV0() (float64, float64) {
	return fc.u[0], fc.v[0]
}

// GetUV1 returns the second set of UV texture coordinates (u1, v1) of the face.
func (fc *Face) GetUV1() (float64, float64) {
	return fc.u[1], fc.v[1]
}

// GetUV2 retrieves the UV texture coordinates (u, v) of the third vertex of the Face.
func (fc *Face) GetUV2() (float64, float64) {
	return fc.u[2], fc.v[2]
}

// GetParent retrieves the parent Sector of the Face instance. Returns nil if no parent is set.
func (fc *Face) GetParent() collisions.IObject {
	return fc.parent
}

// SetParent assigns a parent sector to the segment.
func (fc *Face) SetParent(parent *Volume) {
	fc.parent = parent
}

// GetLabel returns the label associated with the face.
func (fc *Face) GetLabel() string {
	return fc.label
}

// SetLabel sets the label for the face.
func (fc *Face) SetLabel(label string) {
	fc.label = label
}

// GetNormal retrieves the normalized vector components (X, Y, Z) representing the face's orientation in 3D space.
func (fc *Face) GetNormal() (float64, float64, float64) {
	return fc.n.X, fc.n.Y, fc.n.Z
}

// GetNormalAbs returns the absolute values of the normal vector components (X, Y, Z) for the face.
func (fc *Face) GetNormalAbs() (float64, float64, float64) {
	return fc.nAbs.X, fc.nAbs.Y, fc.nAbs.Z
}

// GetMaterial retrieves the material object associated with the Face or nil if no material is set.
func (fc *Face) GetMaterial() *textures.Material {
	return fc.material
}

// ClonePoints returns a copy of the face's three triangle points.
func (fc *Face) ClonePoints() [3]geometry.XYZ {
	return fc.tri.Points
}

// GetPoint0 returns the X, Y, and Z coordinates of the first vertex of the triangle that defines the face.
func (fc *Face) GetPoint0() (float64, float64, float64) {
	return fc.tri.GetPoint0()
}

// GetPoint1 retrieves the coordinates (X, Y, Z) of the second vertex of the triangle defining the Face.
func (fc *Face) GetPoint1() (float64, float64, float64) {
	return fc.tri.GetPoint1()
}

// GetPoint2 returns the X, Y, and Z coordinates of the third vertex in the Face's triangular geometry.
func (fc *Face) GetPoint2() (float64, float64, float64) {
	return fc.tri.GetPoint2()
}

// PointInside2d determines if the provided 2D point (px, py) lies inside the tri defined by the Face's first three points.
func (fc *Face) PointInside2d(px, py float64) bool {
	return fc.tri.PointInside2d(px, py)
}

// Rebuild calculates the axis-aligned bounding box (AABB) for the segment, considering both 2D and 3D cases.
func (fc *Face) Rebuild() {
	fc.computeAABB()
	fc.computeNormal()
	fc.computeUV()
}

// GetAABB returns the axis-aligned bounding box (AABB) associated with the segment.
func (fc *Face) GetAABB() *aabb.AABB {
	return fc.aabb
}

// Zero resets all triangle vertex coordinates of the face to the origin.
func (fc *Face) Zero() {
	fc.tri.Zero()
}

// TransformPoints copies the source face points into this face, offset by the given origin coordinates.
func (fc *Face) TransformPoints(src *Face, originX, originY, originZ float64) {
	fc.tri.TransformPoints(src.tri, originX, originY, originZ)
}

// CopyPoints copies the coordinates of all triangle points from src into fc.
func (fc *Face) CopyPoints(src *Face) {
	fc.tri.CopyPoints(src.tri)
}

func (fc *Face) IsSolid() bool {
	mat := fc.GetMaterial()
	if mat == nil {
		return false
	}
	if mat.GetKind() == int(config.MaterialKindSky) || !mat.IsSolid() {
		//Skybox or non-solid surface: ignore collision and volume tracking
		return false
	}
	return true
}

// computeNormal calculates and assigns the n vector (geometry.XYZ) for the Face based on its points and geometry.
func (fc *Face) computeNormal() {
	const snapToZero = 1e-15
	const areaCulling = 1e-12

	fc.n = geometry.XYZ{X: 0, Y: 0, Z: 1}
	// Prodotto vettoriale standard per poligoni 3D
	p0, p1, p2 := fc.tri.Points[0], fc.tri.Points[1], fc.tri.Points[2]
	v1x, v1y, v1z := p1.X-p0.X, p1.Y-p0.Y, p1.Z-p0.Z
	v2x, v2y, v2z := p2.X-p0.X, p2.Y-p0.Y, p2.Z-p0.Z
	nx := v1y*v2z - v1z*v2y
	ny := v1z*v2x - v1x*v2z
	nz := v1x*v2y - v1y*v2x
	// 1. SNAP TO ZERO: Puliamo il rumore di fondo prima della magnitudo
	if math.Abs(nx) < snapToZero {
		nx = 0
	}
	if math.Abs(ny) < snapToZero {
		ny = 0
	}
	if math.Abs(nz) < snapToZero {
		nz = 0
	}
	l := math.Sqrt(nx*nx + ny*ny + nz*nz)
	// 2. AREA CULLING: Se la magnitudo (area) è troppo piccola,
	// il triangolo è un "Sliver" degenere prodotto dal clipping.
	if l < areaCulling {
		// Lo marchiamo con normale nulla: la QueryCollisionCage lo scarterà
		fc.n.X, fc.n.Y, fc.n.Z = 0, 0, 0
		fc.nAbs.X, fc.nAbs.Y, fc.nAbs.Z = 0, 0, 0
		//fmt.Printf("Ignorato triangolo degenere (Sliver) %f\n", l)
		return
	}
	fc.n.X, fc.n.Y, fc.n.Z = nx/l, ny/l, nz/l
	fc.nAbs.X, fc.nAbs.Y, fc.nAbs.Z = math.Abs(fc.n.X), math.Abs(fc.n.Y), math.Abs(fc.n.Z)
}

// computeAABB calculates the axis-aligned bounding box (AABB) for the Face using its points and optional Z bounds.
func (fc *Face) computeAABB() {
	const eps = 0.001
	minX, minY, minZ, maxX, maxY, maxZ := fc.tri.MinMax()
	if minZ == maxZ {
		minZ -= eps
		maxZ += eps
	}
	fc.aabb.Rebuild(minX-eps, minY-eps, minZ, maxX+eps, maxY+eps, maxZ)
}

// computeUV computes the UV mapping for the current face based on its n, material, and texture scaling factors.
func (fc *Face) computeUV() {
	if fc.lockUV {
		return
	}
	if fc.material == nil {
		return
	}
	tex := fc.material.GetCurrentFrame()
	if tex == nil {
		return
	}
	w, h := tex.GetSize()
	nAbsX, nAbsY, nAbsZ := fc.GetNormalAbs()
	// Pure Triplanar Projection.
	if nAbsZ >= nAbsX && nAbsZ >= nAbsY {
		// Upper / Lower (Floors and Ceilings)
		u0, v0 := fc.tri.Points[0].X/w, fc.tri.Points[0].Y/h
		u1, v1 := fc.tri.Points[1].X/w, fc.tri.Points[1].Y/h
		u2, v2 := fc.tri.Points[2].X/w, fc.tri.Points[2].Y/h
		fc.SetUV(u0, v0, u1, v1, u2, v2)
	} else if nAbsY >= nAbsX && nAbsY >= nAbsZ {
		// Walls facing Y
		fc.SetUV(fc.tri.Points[0].X/w, fc.tri.Points[0].Z/h, fc.tri.Points[1].X/w, fc.tri.Points[1].Z/h, fc.tri.Points[2].X/w, fc.tri.Points[2].Z/h)
	} else {
		// Walls facing X
		fc.SetUV(fc.tri.Points[0].Y/w, fc.tri.Points[0].Z/h, fc.tri.Points[1].Y/w, fc.tri.Points[1].Z/h, fc.tri.Points[2].Y/w, fc.tri.Points[2].Z/h)
	}
}

// SumPoints adds the face's triangle point coordinates to the provided totals and returns the updated sums and count.
func (fc *Face) SumPoints(cx, cy, cz float64, count int) (float64, float64, float64, int) {
	return fc.tri.SumPoints(cx, cy, cz, count)
}

// GetMinMax returns the bounds expanded to include all vertices of the face.
func (fc *Face) GetMinMax(minX, minY, minZ, maxX, maxY, maxZ float64) (float64, float64, float64, float64, float64, float64) {
	return fc.tri.MinMaxExpands(minX, minY, minZ, maxX, maxY, maxZ)
}

/*
func (s *Face) SweepTest(viewX, viewY, viewZ, velX, velY, velZ, eRadX, eRadY, eRadZ float64) (float64, float64, float64, float64, bool) {
	// ==========================================
	// 1. TRASFORMAZIONE E-SPACE ASSOLUTA
	// ==========================================
	eVX, eVY, eVZ := viewX/eRadX, viewY/eRadY, viewZ/eRadZ
	eVelX, eVelY, eVelZ := velX/eRadX, velY/eRadY, velZ/eRadZ

	p0x, p0y, p0z := s.tri[0].X/eRadX, s.tri[0].Y/eRadY, s.tri[0].Z/eRadZ
	p1x, p1y, p1z := s.tri[1].X/eRadX, s.tri[1].Y/eRadY, s.tri[1].Z/eRadZ
	p2x, p2y, p2z := s.tri[2].X/eRadX, s.tri[2].Y/eRadY, s.tri[2].Z/eRadZ

	// Ricalcoliamo la normale RIGOROSAMENTE in e-space basandoci sui vertici deformati.
	// Questo garantisce perfetta coplanarità matematica a prescindere da come
	// il map-exporter aveva salvato s.n.
	e1x, e1y, e1z := p1x-p0x, p1y-p0y, p1z-p0z
	e2x, e2y, e2z := p2x-p0x, p2y-p0y, p2z-p0z

	enX := e1y*e2z - e1z*e2y
	enY := e1z*e2x - e1x*e2z
	enZ := e1x*e2y - e1y*e2x

	lenN := math.Sqrt(enX*enX + enY*enY + enZ*enZ)
	if lenN < 1e-8 {
		return 0, 0, 0, 0, false // Scarta poligoni degeneri a scala microscopica
	}
	enX, enY, enZ = enX/lenN, enY/lenN, enZ/lenN

	// ==========================================
	// 2. GESTIONE UNIVERSALE DOUBLE-SIDED
	// ==========================================
	// Qualunque sia l'orientamento originale del triangolo, per calcolare l'impatto
	// forziamo il piano matematico a opporsi alla velocità dell'entità.
	// Nessun vertex flip. Cramer's Rule è immune a questa inversione.
	vDotN := eVelX*enX + eVelY*enY + eVelZ*enZ

	if vDotN > 0.0 {
		enX, enY, enZ = -enX, -enY, -enZ
		vDotN = -vDotN
	}

	const radius = 1.0
	const epsilon = 1e-4 // TOI Back-off

	distStart := (eVX-p0x)*enX + (eVY-p0y)*enY + (eVZ-p0z)*enZ

	// Culling dinamico: scartiamo solo se l'entità è GIA' affondata oltre il proprio raggio
	// rispetto al lato che stiamo colpendo.
	if distStart < -radius {
		return 0, 0, 0, 0, false
	}

	var minT = 1.0
	var hit = false
	var cNx, cNy, cNz float64

	// ==========================================
	// FASE A: IMPATTO SUL PIANO (Slab Intersection)
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
		tPlane := max(0.0, t0) // Micro-penetration clamp

		if tPlane <= 1.0 {
			cx := eVX + eVelX*tPlane
			cy := eVY + eVelY*tPlane
			cz := eVZ + eVelZ*tPlane

			hX := cx - enX*radius
			hY := cy - enY*radius
			hZ := cz - enZ*radius

			vX, vY, vZ := hX-p0x, hY-p0y, hZ-p0z

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

				// Tolleranza per fessure a scala sub-millimetrica
				const baryEps = -1e-3
				if v >= baryEps && w >= baryEps && u >= baryEps {
					safeT := max(0.0, tPlane-epsilon)
					hit = true
					minT = safeT
					cNx, cNy, cNz = enX, enY, enZ
					goto RESOLVE
				}
			}
		}
	}

	// ==========================================
	// FASE B: SWEEP CONTINUO SU VERTICI E SPIGOLI
	// ==========================================
	{
		velSq := eVelX*eVelX + eVelY*eVelY + eVelZ*eVelZ
		if velSq < 1e-8 {
			goto RESOLVE
		}

		solveQuad := func(a, b, c float64) (float64, bool) {
			if c <= 0.0 {
				// Anti-Tunneling: blocca solo se premiamo attivamente contro la geometria (b < 0)
				if b < -1e-6 {
					return 0.0, true
				}
				return 1.0, false
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

		pts := [3][3]float64{{p0x, p0y, p0z}, {p1x, p1y, p1z}, {p2x, p2y, p2z}}
		for _, p := range pts {
			vx, vy, vz := eVX-p[0], eVY-p[1], eVZ-p[2]
			a := velSq
			b := 2.0 * (eVelX*vx + eVelY*vy + eVelZ*vz)
			c := (vx*vx + vy*vy + vz*vz) - radius*radius

			if t, ok := solveQuad(a, b, c); ok {
				minT = t
				hit = true
				cNx, cNy, cNz = eVX+eVelX*t-p[0], eVY+eVelY*t-p[1], eVZ+eVelZ*t-p[2]
			}
		}

		for i := 0; i < 3; i++ {
			pA := pts[i]
			pB := pts[(i+1)%3]

			edgeX, edgeY, edgeZ := pB[0]-pA[0], pB[1]-pA[1], pB[2]-pA[2]
			edgeLenSq := edgeX*edgeX + edgeY*edgeY + edgeZ*edgeZ

			if edgeLenSq < 1e-8 {
				continue
			}

			vx, vy, vz := eVX-pA[0], eVY-pA[1], eVZ-pA[2]

			edgeDotVel := eVelX*edgeX + eVelY*edgeY + eVelZ*edgeZ
			edgeDotOrig := vx*edgeX + vy*edgeY + vz*edgeZ

			a := edgeLenSq*velSq - edgeDotVel*edgeDotVel
			b := edgeLenSq*2.0*(eVelX*vx+eVelY*vy+eVelZ*vz) - 2.0*edgeDotVel*edgeDotOrig
			c := edgeLenSq*((vx*vx+vy*vy+vz*vz)-radius*radius) - edgeDotOrig*edgeDotOrig

			if a == 0.0 {
				continue
			}

			if t, ok := solveQuad(a, b, c); ok {
				f := (edgeDotOrig + edgeDotVel*t) / edgeLenSq
				if f >= 0.0 && f <= 1.0 {
					minT = t
					hit = true
					closestX := pA[0] + f*edgeX
					closestY := pA[1] + f*edgeY
					closestZ := pA[2] + f*edgeZ
					cNx, cNy, cNz = eVX+eVelX*t-closestX, eVY+eVelY*t-closestY, eVZ+eVelZ*t-closestZ
				}
			}
		}
	}

RESOLVE:
	if hit {
		safeT := max(0.0, minT-epsilon)

		// 3. RIPRISTINO SCALA EUCLIDEA
		realNx := cNx / eRadX
		realNy := cNy / eRadY
		realNz := cNz / eRadZ

		l := math.Sqrt(realNx*realNx + realNy*realNy + realNz*realNz)
		if l > 1e-8 {
			realNx, realNy, realNz = realNx/l, realNy/l, realNz/l
		} else {
			realNx = enX / eRadX
			realNy = enY / eRadY
			realNz = enZ / eRadZ
			l = math.Sqrt(realNx*realNx + realNy*realNy + realNz*realNz)
			if l > 1e-8 {
				realNx, realNy, realNz = realNx/l, realNy/l, realNz/l
			}
		}

		// 4. FILTRO ANTI-HIJACKING (Impedisce alle superfici parallele di bloccare lo sweep)
		dotVelNorm := velX*realNx + velY*realNy + velZ*realNz
		if dotVelNorm >= -1e-5 {
			return 0, 0, 0, 0, false
		}

		return safeT, realNx, realNy, realNz, true
	}

	return 0, 0, 0, 0, false
}

*/

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
