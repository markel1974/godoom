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
	id          int
	parent      *Volume
	label       string
	aabb        *aabb.AABB
	tri         *geometry.Triangle3D
	n           geometry.XYZ
	nAbs        geometry.XYZ
	material    *textures.Material
	u           [3]float64
	v           [3]float64
	lockUV      bool
	singleSided bool
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

// Set0 sets the coordinates of the first point in the face.
func (fc *Face) Set0(x, y, z float64) {
	fc.tri.Points[0].X = x
	fc.tri.Points[0].Y = y
	fc.tri.Points[0].Z = z
}

// Set1 updates the second point of the face to the given x, y, and z coordinates.
func (fc *Face) Set1(x, y, z float64) {
	fc.tri.Points[1].X = x
	fc.tri.Points[1].Y = y
	fc.tri.Points[1].Z = z
}

// Set2 updates the coordinates of the face's third vertex.
func (fc *Face) Set2(x, y, z float64) {
	fc.tri.Points[2].X = x
	fc.tri.Points[2].Y = y
	fc.tri.Points[2].Z = z
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

// SetU sets the U coordinates for the face vertices.
func (fc *Face) SetU(u0, u1, u2 float64) {
	fc.u[0] = u0
	fc.u[1] = u1
	fc.u[2] = u2
}

// SetV sets the face's V components to v0, v1, and v2.
func (fc *Face) SetV(v0, v1, v2 float64) {
	fc.v[0] = v0
	fc.v[1] = v1
	fc.v[2] = v2
}

func (fc *Face) GetUVFlat() (float64, float64, float64, float64, float64, float64) {
	return fc.u[0], fc.u[1], fc.u[2], fc.v[0], fc.v[1], fc.v[2]
}

// GetUV retrieves the `u` and `v` coordinate arrays representing the UV mapping of the face.
func (fc *Face) GetUV() ([3]float64, [3]float64) {
	return fc.u, fc.v
}

func (fc *Face) GetU() (float64, float64, float64) {
	return fc.u[0], fc.u[1], fc.u[2]
}

func (fc *Face) GetV() (float64, float64, float64) {
	return fc.v[0], fc.v[1], fc.v[2]
}

/*
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

*/

// GetParent retrieves the parent Sector of the Face instance. Returns nil if no parent is set.
func (fc *Face) GetParent() *collisions.Solver {
	return fc.parent.GetSolver()
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

// SetMaterial assigns the material used by the face.
func (fc *Face) SetMaterial(mat *textures.Material) {
	fc.material = mat
}

// ClonePoints returns a copy of the face's three triangle points.
func (fc *Face) ClonePoints() [3]geometry.XYZ {
	return fc.tri.Points
}

func (fc *Face) GetPointsFlat() (float64, float64, float64, float64, float64, float64, float64, float64, float64) {
	return fc.tri.GetPointsFlat()
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

// SetSingleSided marks whether collisions only block from the face's front normal side.
func (fc *Face) SetSingleSided(singleSided bool) {
	fc.singleSided = singleSided
}

// IsSingleSided reports whether collisions must ignore approaches behind the face plane.
func (fc *Face) IsSingleSided() bool {
	return fc.singleSided
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
