package volumes

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
	"github.com/markel1974/godoom/mr_tech/textures"
)

const (
	// EdgeABActive marks triangle edge p0->p1 as an active convex/boundary collision edge.
	EdgeABActive uint8 = 1 << 0
	// EdgeBCActive marks triangle edge p1->p2 as an active convex/boundary collision edge.
	EdgeBCActive uint8 = 1 << 1
	// EdgeCAActive marks triangle edge p2->p0 as an active convex/boundary collision edge.
	EdgeCAActive uint8 = 1 << 2
	// EdgeAllActive marks all three triangle edges as active collision edges.
	EdgeAllActive = EdgeABActive | EdgeBCActive | EdgeCAActive
)

// Face represents a boundary or edge of a Sector, defined by its geometry, connectivity, and optional metadata.
type Face struct {
	id                 int
	parent             *Volume
	material           *textures.Material
	aabb               aabb.AABB
	tri                geometry.Triangle3D32
	n                  geometry.XYZ32
	nAbs               geometry.XYZ32
	vertexNormals      [3]geometry.XYZ32
	explicitNormals    [3]geometry.XYZ32
	tangents           [3]geometry.XYZ32
	tangentSigns       [3]float32
	u                  [3]float32
	v                  [3]float32
	lmU                [3]float32
	lmV                [3]float32
	lightmapID         int32
	hasExplicitNormals bool
	lockUV             bool
	singleSided        bool
	activeEdges        uint8
}

func toTri32(tri [3]geometry.XYZ) geometry.Triangle3D32 {
	return geometry.Triangle3D32{
		Points: [3]geometry.XYZ32{
			tri[0].To32(),
			tri[1].To32(),
			tri[2].To32(),
		},
	}
}

func toFloat32Triple(a, b, c float64) [3]float32 {
	return [3]float32{float32(a), float32(b), float32(c)}
}

// NewFaceEmpty returns a Face with an empty triangle, the given material, and UV coordinates.
func NewFaceEmpty(material *textures.Material, u, v [3]float64) *Face {
	out := &Face{
		material:     material,
		aabb:         aabb.MakeAABB(),
		u:            [3]float32{float32(u[0]), float32(u[1]), float32(u[2])},
		v:            [3]float32{float32(v[0]), float32(v[1]), float32(v[2])},
		lightmapID:   -1,
		tangentSigns: [3]float32{1, 1, 1},
		activeEdges:  EdgeAllActive,
	}
	return out
}

// NewFace creates a new 3D segment with specified points and material, and computes its normal and AABB.
func NewFace(tri [3]geometry.XYZ, material *textures.Material) *Face {
	out := &Face{
		material:    material,
		aabb:        aabb.MakeAABB(),
		tri:         toTri32(tri),
		lightmapID:  -1,
		activeEdges: EdgeAllActive,
	}
	out.tangentSigns[0] = 1
	out.tangentSigns[1] = 1
	out.tangentSigns[2] = 1
	out.Rebuild()
	return out
}

// NewFaceSubdivided returns a new face for tri, preserving metadata and applying UV coordinates when UVs are locked.
func (fc *Face) NewFaceSubdivided(tri [3]geometry.XYZ, uvU, uvV [3]float64) *Face {
	out := NewFace(tri, fc.material)
	out.parent = fc.parent
	out.singleSided = fc.singleSided
	out.lightmapID = fc.lightmapID
	out.lmU = fc.lmU
	out.lmV = fc.lmV
	out.lockUV = fc.lockUV
	if fc.lockUV {
		out.SetUV(uvU[0], uvV[0], uvU[1], uvV[1], uvU[2], uvV[2])
	}
	return out
}

// GetTri returns the three float32 vertices that define the face triangle.
func (fc *Face) GetTri() *geometry.Triangle3D32 {
	return &fc.tri
}

// Get0 returns the x, y, and z coordinates of the face's first vertex in float64.
func (fc *Face) Get0() (float64, float64, float64) {
	p := &fc.tri.Points[0]
	return float64(p.X), float64(p.Y), float64(p.Z)
}

// Get1 returns the x, y, and z coordinates of the face's second triangle point in float64.
func (fc *Face) Get1() (float64, float64, float64) {
	p := &fc.tri.Points[1]
	return float64(p.X), float64(p.Y), float64(p.Z)
}

// Get2 returns the coordinates of the face's third point in float64.
func (fc *Face) Get2() (float64, float64, float64) {
	p := &fc.tri.Points[2]
	return float64(p.X), float64(p.Y), float64(p.Z)
}

// Set0 sets the coordinates of the first point in the face.
func (fc *Face) Set0(x, y, z float64) {
	fc.tri.Points[0].X = float32(x)
	fc.tri.Points[0].Y = float32(y)
	fc.tri.Points[0].Z = float32(z)
}

// Set1 updates the second point of the face to the given x, y, and z coordinates.
func (fc *Face) Set1(x, y, z float64) {
	fc.tri.Points[1].X = float32(x)
	fc.tri.Points[1].Y = float32(y)
	fc.tri.Points[1].Z = float32(z)
}

// Set2 updates the coordinates of the face's third vertex.
func (fc *Face) Set2(x, y, z float64) {
	fc.tri.Points[2].X = float32(x)
	fc.tri.Points[2].Y = float32(y)
	fc.tri.Points[2].Z = float32(z)
}

// Translate shifts the coordinates of a triangle vertex by specified offsets in X, Y, and Z directions.
func (fc *Face) Translate(idx int, offX, offY, offZ float64) (float64, float64, float64) {
	p := &fc.tri.Points[idx]
	return float64(p.X) + offX, float64(p.Y) + offY, float64(p.Z) + offZ
}

// GetUV returns the U and V texture coordinate arrays for the face vertices as float64.
func (fc *Face) GetUV() ([3]float64, [3]float64) {
	return [3]float64{float64(fc.u[0]), float64(fc.u[1]), float64(fc.u[2])},
		[3]float64{float64(fc.v[0]), float64(fc.v[1]), float64(fc.v[2])}
}

// GetU returns the three U texture coordinates for the face vertices.
func (fc *Face) GetU() (float64, float64, float64) {
	return float64(fc.u[0]), float64(fc.u[1]), float64(fc.u[2])
}

// GetV returns the V texture coordinates for the face's three vertices.
func (fc *Face) GetV() (float64, float64, float64) {
	return float64(fc.v[0]), float64(fc.v[1]), float64(fc.v[2])
}

// GetUVFlat returns the face texture coordinates as six float32 values for presentation.
func (fc *Face) GetUVFlat() (float32, float32, float32, float32, float32, float32) {
	return fc.u[0], fc.u[1], fc.u[2], fc.v[0], fc.v[1], fc.v[2]
}

// GetVertexNormalsFlat returns the three vertex normal vectors as nine float32 components for presentation.
func (fc *Face) GetVertexNormalsFlat() (float32, float32, float32, float32, float32, float32, float32, float32, float32) {
	ptr0 := &fc.vertexNormals[0]
	ptr1 := &fc.vertexNormals[1]
	ptr2 := &fc.vertexNormals[2]
	return ptr0.X, ptr0.Y, ptr0.Z, ptr1.X, ptr1.Y, ptr1.Z, ptr2.X, ptr2.Y, ptr2.Z
}

// GetTangentsFlat returns per-vertex tangent XYZ + bitangent sign W for all three vertices in float32 for presentation.
func (fc *Face) GetTangentsFlat() (float32, float32, float32, float32, float32, float32, float32, float32, float32, float32, float32, float32) {
	ptr0 := &fc.tangents[0]
	ptr1 := &fc.tangents[1]
	ptr2 := &fc.tangents[2]
	return ptr0.X, ptr0.Y, ptr0.Z, fc.tangentSigns[0], ptr1.X, ptr1.Y, ptr1.Z, fc.tangentSigns[1], ptr2.X, ptr2.Y, ptr2.Z, fc.tangentSigns[2]
}

// GetEdgePoints returns the two vertices for edgeIdx and the triangle vertex opposite that edge in float32.
func (fc *Face) GetEdgePoints(edgeIdx int) (pA, pB, opp geometry.XYZ32) {
	pts := &fc.tri.Points
	switch edgeIdx {
	case 0:
		return pts[0], pts[1], pts[2]
	case 1:
		return pts[1], pts[2], pts[0]
	default:
		return pts[2], pts[0], pts[1]
	}
}

// LockUV locks or unlocks the UV coordinates of a Face based on the provided lockUV parameter.
func (fc *Face) LockUV(lockUV bool) {
	fc.lockUV = lockUV
}

// SetUV sets the UV texture coordinates for the three vertices of the face.
func (fc *Face) SetUV(u0, v0, u1, v1, u2, v2 float64) {
	fc.u[0], fc.v[0] = float32(u0), float32(v0)
	fc.u[1], fc.v[1] = float32(u1), float32(v1)
	fc.u[2], fc.v[2] = float32(u2), float32(v2)
	fc.computeTangent()
}

// SetU sets the U coordinates for the face vertices.
func (fc *Face) SetU(u0, u1, u2 float64) {
	fc.u[0], fc.u[1], fc.u[2] = float32(u0), float32(u1), float32(u2)
}

// SetV sets the face's V components to v0, v1, and v2.
func (fc *Face) SetV(v0, v1, v2 float64) {
	fc.v[0], fc.v[1], fc.v[2] = float32(v0), float32(v1), float32(v2)
}

// SetVertexNormals sets explicit per-vertex normals (preserving them across Rebuild/smoothing) and updates tangents.
func (fc *Face) SetVertexNormals(n0X, n0Y, n0Z, n1X, n1Y, n1Z, n2X, n2Y, n2Z float64) {
	fX, fY, fZ := fc.n.X, fc.n.Y, fc.n.Z
	ptr0 := &fc.explicitNormals[0]
	ptr0.X, ptr0.Y, ptr0.Z = NormalizeXYZ64(n0X, n0Y, n0Z, fX, fY, fZ)
	ptr1 := &fc.explicitNormals[1]
	ptr1.X, ptr1.Y, ptr1.Z = NormalizeXYZ64(n1X, n1Y, n1Z, fX, fY, fZ)
	ptr2 := &fc.explicitNormals[2]
	ptr2.X, ptr2.Y, ptr2.Z = NormalizeXYZ64(n2X, n2Y, n2Z, fX, fY, fZ)
	fc.vertexNormals = fc.explicitNormals
	fc.hasExplicitNormals = true
	fc.computeTangent()
}

// SetComputedVertexNormals assigns computed/transformed per-vertex normals without marking them as immutable artist normals.
func (fc *Face) SetComputedVertexNormals(n0X, n0Y, n0Z, n1X, n1Y, n1Z, n2X, n2Y, n2Z float64) {
	fX, fY, fZ := fc.n.X, fc.n.Y, fc.n.Z
	ptr0 := &fc.vertexNormals[0]
	ptr0.X, ptr0.Y, ptr0.Z = NormalizeXYZ64(n0X, n0Y, n0Z, fX, fY, fZ)
	ptr1 := &fc.vertexNormals[1]
	ptr1.X, ptr1.Y, ptr1.Z = NormalizeXYZ64(n1X, n1Y, n1Z, fX, fY, fZ)
	ptr2 := &fc.vertexNormals[2]
	ptr2.X, ptr2.Y, ptr2.Z = NormalizeXYZ64(n2X, n2Y, n2Z, fX, fY, fZ)
}

// HasExplicitNormals reports whether explicit per-vertex normals were provided.
func (fc *Face) HasExplicitNormals() bool {
	return fc.hasExplicitNormals
}

// SetTangents sets per-vertex tangents and handedness signs directly (e.g. for rotated clones).
func (fc *Face) SetTangents(t0X, t0Y, t0Z, t1X, t1Y, t1Z, t2X, t2Y, t2Z float64, s0, s1, s2 float64) {
	ptr0 := &fc.tangents[0]
	ptr0.X, ptr0.Y, ptr0.Z = float32(t0X), float32(t0Y), float32(t0Z)
	ptr1 := &fc.tangents[1]
	ptr1.X, ptr1.Y, ptr1.Z = float32(t1X), float32(t1Y), float32(t1Z)
	ptr2 := &fc.tangents[2]
	ptr2.X, ptr2.Y, ptr2.Z = float32(t2X), float32(t2Y), float32(t2Z)
	fc.tangentSigns[0], fc.tangentSigns[1], fc.tangentSigns[2] = float32(s0), float32(s1), float32(s2)
}

// SetLightmap assigns the baked lightmap texture layer index and UV coordinates per vertex.
func (fc *Face) SetLightmap(id int32, u0, v0, u1, v1, u2, v2 float64) {
	fc.lightmapID = id
	fc.lmU[0], fc.lmV[0] = float32(u0), float32(v0)
	fc.lmU[1], fc.lmV[1] = float32(u1), float32(v1)
	fc.lmU[2], fc.lmV[2] = float32(u2), float32(v2)
}

// SetLightmapArrays assigns lightmap layer index and U/V coordinate arrays.
func (fc *Face) SetLightmapArrays(id int32, u, v [3]float64) {
	fc.lightmapID = id
	fc.lmU = toFloat32Triple(u[0], u[1], u[2])
	fc.lmV = toFloat32Triple(v[0], v[1], v[2])
}

// GetLightmap returns the baked lightmap layer index (-1 if none) and vertex U/V arrays in float32 for presentation.
func (fc *Face) GetLightmap() (int32, [3]float32, [3]float32) {
	return fc.lightmapID, fc.lmU, fc.lmV
}

// CopyRenderAttributes copies normals, tangents, and lightmap coordinates from src to fc without allocations.
func (fc *Face) CopyRenderAttributes(src *Face) {
	fc.n = src.n
	fc.nAbs = src.nAbs
	fc.vertexNormals = src.vertexNormals
	fc.explicitNormals = src.explicitNormals
	fc.hasExplicitNormals = src.hasExplicitNormals
	fc.tangents = src.tangents
	fc.tangentSigns = src.tangentSigns
	fc.lmU = src.lmU
	fc.lmV = src.lmV
	fc.lightmapID = src.lightmapID
}

// EnsureOutwardWinding ensures triangle winding is CCW relative to expectedNormal.
// If the face normal points opposite expectedNormal, vertices 1 and 2 (and attributes) are swapped.
func (fc *Face) EnsureOutwardWinding(expectedNormal geometry.XYZ) bool {
	p0, p1, p2 := fc.tri.Points[0], fc.tri.Points[1], fc.tri.Points[2]
	v1x, v1y, v1z := float64(p1.X)-float64(p0.X), float64(p1.Y)-float64(p0.Y), float64(p1.Z)-float64(p0.Z)
	v2x, v2y, v2z := float64(p2.X)-float64(p0.X), float64(p2.Y)-float64(p0.Y), float64(p2.Z)-float64(p0.Z)
	nx := v1y*v2z - v1z*v2y
	ny := v1z*v2x - v1x*v2z
	nz := v1x*v2y - v1y*v2x
	if nx*expectedNormal.X+ny*expectedNormal.Y+nz*expectedNormal.Z < 0 {
		fc.tri.Points[1], fc.tri.Points[2] = fc.tri.Points[2], fc.tri.Points[1]
		fc.u[1], fc.u[2] = fc.u[2], fc.u[1]
		fc.v[1], fc.v[2] = fc.v[2], fc.v[1]
		fc.lmU[1], fc.lmU[2] = fc.lmU[2], fc.lmU[1]
		fc.lmV[1], fc.lmV[2] = fc.lmV[2], fc.lmV[1]
		fc.vertexNormals[1], fc.vertexNormals[2] = fc.vertexNormals[2], fc.vertexNormals[1]
		fc.explicitNormals[1], fc.explicitNormals[2] = fc.explicitNormals[2], fc.explicitNormals[1]
		fc.computeNormal()
		fc.computeTangent()
		return true
	}
	return false
}

// GetParent retrieves the parent Sector of the Face instance. Returns nil if no parent is set.
func (fc *Face) GetParent() *collisions.Solver {
	return fc.parent.GetSolver()
}

// SetParent assigns a parent sector to the segment.
func (fc *Face) SetParent(parent *Volume) {
	fc.parent = parent
}

// GetNormal retrieves the normalized vector components (X, Y, Z) representing the face's orientation in float64.
func (fc *Face) GetNormal() (float64, float64, float64) {
	return float64(fc.n.X), float64(fc.n.Y), float64(fc.n.Z)
}

// GetNormalAbs returns the absolute values of the normal vector components (X, Y, Z) for the face in float64.
func (fc *Face) GetNormalAbs() (float64, float64, float64) {
	return float64(fc.nAbs.X), float64(fc.nAbs.Y), float64(fc.nAbs.Z)
}

// GetMaterial retrieves the material object associated with the Face or nil if no material is set.
func (fc *Face) GetMaterial() *textures.Material {
	return fc.material
}

// SetMaterial assigns the material used by the face.
func (fc *Face) SetMaterial(mat *textures.Material) {
	fc.material = mat
}

// ClonePoints returns a copy of the face's three triangle points in float64.
func (fc *Face) ClonePoints() [3]geometry.XYZ {
	p0, p1, p2 := fc.tri.Points[0], fc.tri.Points[1], fc.tri.Points[2]
	return [3]geometry.XYZ{
		{X: float64(p0.X), Y: float64(p0.Y), Z: float64(p0.Z)},
		{X: float64(p1.X), Y: float64(p1.Y), Z: float64(p1.Z)},
		{X: float64(p2.X), Y: float64(p2.Y), Z: float64(p2.Z)},
	}
}

// GetPointsFlat returns the face triangle's three points as nine flat float32 coordinate values for presentation.
func (fc *Face) GetPointsFlat() (float32, float32, float32, float32, float32, float32, float32, float32, float32) {
	return fc.tri.GetPointsFlat()
}

// GetPoint0 returns the X, Y, and Z coordinates of the first vertex of the triangle in float64.
func (fc *Face) GetPoint0() (float64, float64, float64) {
	return fc.Get0()
}

// GetPoint1 retrieves the coordinates (X, Y, Z) of the second vertex of the triangle in float64.
func (fc *Face) GetPoint1() (float64, float64, float64) {
	return fc.Get1()
}

// GetPoint2 returns the X, Y, and Z coordinates of the third vertex in the Face's triangle in float64.
func (fc *Face) GetPoint2() (float64, float64, float64) {
	return fc.Get2()
}

// PointInside2d determines if the provided 2D point (px, py) lies inside the tri defined by the Face's first three points.
func (fc *Face) PointInside2d(px, py float64) bool {
	p0x, p0y, _ := fc.Get0()
	p1x, p1y, _ := fc.Get1()
	p2x, p2y, _ := fc.Get2()
	d1 := (px-p0x)*(p1y-p0y) - (py-p0y)*(p1x-p0x)
	d2 := (px-p1x)*(p2y-p1y) - (py-p1y)*(p2x-p1x)
	d3 := (px-p2x)*(p0y-p2y) - (py-p2y)*(p0x-p2x)
	const eps = -0.001
	hasNeg := (d1 < eps) || (d2 < eps) || (d3 < eps)
	hasPos := (d1 > -eps) || (d2 > -eps) || (d3 > -eps)
	return !(hasNeg && hasPos)
}

// IntersectRay calculates the intersection distance of a ray with the triangle face (Möller–Trumbore in float64).
// Returns (distance, true) if the ray hits within maxDist.
func (fc *Face) IntersectRay(oX, oY, oZ, dirX, dirY, dirZ, maxDist float64) (float64, bool) {
	const eps = 1e-7
	p0x, p0y, p0z := fc.Get0()
	p1x, p1y, p1z := fc.Get1()
	p2x, p2y, p2z := fc.Get2()

	e1x, e1y, e1z := p1x-p0x, p1y-p0y, p1z-p0z
	e2x, e2y, e2z := p2x-p0x, p2y-p0y, p2z-p0z

	// h = dir x e2
	hx := dirY*e2z - dirZ*e2y
	hy := dirZ*e2x - dirX*e2z
	hz := dirX*e2y - dirY*e2x

	// a = e1 . h
	a := e1x*hx + e1y*hy + e1z*hz
	if fc.singleSided {
		if a < eps {
			return 0, false
		}
	} else {
		if math.Abs(a) < eps {
			return 0, false
		}
	}

	f := 1.0 / a
	// s = origin - p0
	sx := oX - p0x
	sy := oY - p0y
	sz := oZ - p0z

	// u = f * (s . h)
	u := f * (sx*hx + sy*hy + sz*hz)
	if u < 0.0 || u > 1.0 {
		return 0, false
	}

	// q = s x e1
	qx := sy*e1z - sz*e1y
	qy := sz*e1x - sx*e1z
	qz := sx*e1y - sy*e1x

	// v = f * (dir . q)
	v := f * (dirX*qx + dirY*qy + dirZ*qz)
	if v < 0.0 || (u+v) > 1.0 {
		return 0, false
	}

	// t = f * (e2 . q)
	t := f * (e2x*qx + e2y*qy + e2z*qz)
	if t > eps && t <= maxDist {
		return t, true
	}

	return 0, false
}

// Rebuild calculates the axis-aligned bounding box (AABB) for the segment, considering both 2D and 3D cases.
func (fc *Face) Rebuild() {
	fc.computeAABB()
	fc.computeNormal()
	fc.computeUV()
	if fc.hasExplicitNormals {
		fc.vertexNormals = fc.explicitNormals
	} else {
		fc.vertexNormals = [3]geometry.XYZ32{fc.n, fc.n, fc.n}
	}
	fc.computeTangent()
}

// InitTemplateSource resets an existing Face in place to build a frame template without reallocating Face structs.
func (fc *Face) InitTemplateSource(tri [3]geometry.XYZ, mat *textures.Material, u, v [3]float64, hasNormals bool, n0, n1, n2 geometry.XYZ) {
	fc.tri = toTri32(tri)
	fc.material = mat
	fc.u = toFloat32Triple(u[0], u[1], u[2])
	fc.v = toFloat32Triple(v[0], v[1], v[2])
	fc.lockUV = true
	fc.lightmapID = -1
	fc.singleSided = false
	fc.activeEdges = EdgeAllActive
	if hasNormals {
		fc.computeNormal()
		fX, fY, fZ := fc.n.X, fc.n.Y, fc.n.Z
		ptr0 := &fc.explicitNormals[0]
		ptr1 := &fc.explicitNormals[1]
		ptr2 := &fc.explicitNormals[2]
		ptr0.X, ptr0.Y, ptr0.Z = NormalizeXYZ64(n0.X, n0.Y, n0.Z, fX, fY, fZ)
		ptr1.X, ptr1.Y, ptr1.Z = NormalizeXYZ64(n1.X, n1.Y, n1.Z, fX, fY, fZ)
		ptr2.X, ptr2.Y, ptr2.Z = NormalizeXYZ64(n2.X, n2.Y, n2.Z, fX, fY, fZ)
		fc.vertexNormals = fc.explicitNormals
		fc.hasExplicitNormals = true
	} else {
		fc.hasExplicitNormals = false
	}
}

// ApplyTemplateGeometry updates triangle vertices, normals, and active edges from a precomputed frame template
// and recomputes derived bounds, plane normal, and tangents without extra allocations.
func (fc *Face) ApplyTemplateGeometry(p0, p1, p2, n0, n1, n2 geometry.XYZ32, activeEdges uint8) {
	fc.tri.Points[0] = p0
	fc.tri.Points[1] = p1
	fc.tri.Points[2] = p2
	fc.computeAABB()
	fc.computeNormal()
	fc.explicitNormals[0] = n0
	fc.explicitNormals[1] = n1
	fc.explicitNormals[2] = n2
	fc.vertexNormals = fc.explicitNormals
	fc.hasExplicitNormals = true
	fc.activeEdges = activeEdges & EdgeAllActive
	fc.computeTangent()
}

// GetAABB returns the axis-aligned bounding box (AABB) associated with the segment.
func (fc *Face) GetAABB() *aabb.AABB {
	return &fc.aabb
}

// Zero resets all triangle vertex coordinates of the face to the origin.
func (fc *Face) Zero() {
	fc.tri.Zero()
}

// TransformPoints copies the source face points into this face, offset by the given float64 origin coordinates.
func (fc *Face) TransformPoints(src *Face, originX, originY, originZ float64) {
	for i := range geometry.Triangle3DPoints32 {
		sp := &src.tri.Points[i]
		dp := &fc.tri.Points[i]
		dp.X = float32(float64(sp.X) + originX)
		dp.Y = float32(float64(sp.Y) + originY)
		dp.Z = float32(float64(sp.Z) + originZ)
	}
}

// CopyPoints copies the coordinates of all triangle points from src into fc.
func (fc *Face) CopyPoints(src *Face) {
	fc.tri.CopyPoints(&src.tri)
}

// SetSingleSided marks whether collisions only block from the face's front normal side.
func (fc *Face) SetSingleSided(singleSided bool) {
	fc.singleSided = singleSided
}

// IsSingleSided reports whether collisions must ignore approaches behind the face plane.
func (fc *Face) IsSingleSided() bool {
	return fc.singleSided
}

// ActiveEdgeMask returns the 3-bit active edge mask (EdgeABActive, EdgeBCActive, EdgeCAActive).
func (fc *Face) ActiveEdgeMask() uint8 {
	return fc.activeEdges
}

// IsEdgeActive reports whether edgeIdx (0=AB, 1=BC, 2=CA) is marked as an active convex/boundary edge.
func (fc *Face) IsEdgeActive(edgeIdx int) bool {
	if edgeIdx < 0 || edgeIdx > 2 {
		return false
	}
	return (fc.activeEdges & (uint8(1) << uint(edgeIdx))) != 0
}

// SetActiveEdgeMask sets the active edge bitmask for the three triangle edges.
func (fc *Face) SetActiveEdgeMask(mask uint8) {
	fc.activeEdges = mask & EdgeAllActive
}

// SetEdgeActive enables or disables active collision handling on edgeIdx (0=AB, 1=BC, 2=CA).
func (fc *Face) SetEdgeActive(edgeIdx int, active bool) {
	if edgeIdx < 0 || edgeIdx > 2 {
		return
	}
	bit := uint8(1) << uint(edgeIdx)
	if active {
		fc.activeEdges |= bit
	} else {
		fc.activeEdges &^= bit
	}
}

// IsSolid reports whether the face has a non-sky material that is solid.
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

// IsDeformable reports whether the face material allows plastic deformation from impacts.
func (fc *Face) IsDeformable() bool {
	if !fc.IsSolid() {
		return false
	}
	mat := fc.GetMaterial()
	if mat == nil {
		return true
	}
	kind := config.MaterialKind(mat.GetKind())
	if kind == config.MaterialKindSky || kind == config.MaterialKindWater ||
		kind == config.MaterialKindSea || kind == config.MaterialKindLava ||
		kind == config.MaterialKindSlime || kind == config.MaterialKindBlood ||
		kind == config.MaterialKindFog {
		return false
	}
	if mat.HasSurfaceParm("nodeform") || mat.HasSurfaceParm("nodamage") {
		return false
	}
	return true
}

// Barycentric computes the 3D barycentric coordinates of point (pX, pY, pZ) relative to the face triangle using float64 precision.
// Returns (w0, w1, w2, inside) where w0+w1+w2=1 and inside is true if p is on or near the triangle surface.
func (fc *Face) Barycentric(pX, pY, pZ float64) (float64, float64, float64, bool) {
	p0x, p0y, p0z := fc.Get0()
	p1x, p1y, p1z := fc.Get1()
	p2x, p2y, p2z := fc.Get2()

	v0x, v0y, v0z := p1x-p0x, p1y-p0y, p1z-p0z
	v1x, v1y, v1z := p2x-p0x, p2y-p0y, p2z-p0z
	v2x, v2y, v2z := pX-p0x, pY-p0y, pZ-p0z

	d00 := v0x*v0x + v0y*v0y + v0z*v0z
	d01 := v0x*v1x + v0y*v1y + v0z*v1z
	d11 := v1x*v1x + v1y*v1y + v1z*v1z
	d20 := v2x*v0x + v2y*v0y + v2z*v0z
	d21 := v2x*v1x + v2y*v1y + v2z*v1z

	denom := d00*d11 - d01*d01
	if math.Abs(denom) < 1e-12 {
		return 0, 0, 0, false
	}

	v := (d11*d20 - d01*d21) / denom
	w := (d00*d21 - d01*d20) / denom
	u := 1.0 - v - w

	const eps = -0.05
	inside := u >= eps && v >= eps && w >= eps
	w0 := math.Max(0, u)
	w1 := math.Max(0, v)
	w2 := math.Max(0, w)
	sum := w0 + w1 + w2
	if sum > 0 {
		w0 /= sum
		w1 /= sum
		w2 /= sum
	}
	return w0, w1, w2, inside
}

// SumPoints adds the face's triangle point coordinates to the provided totals in float64 and returns the updated sums and count.
func (fc *Face) SumPoints(cx, cy, cz float64, count int) (float64, float64, float64, int) {
	for i := range geometry.Triangle3DPoints32 {
		p := &fc.tri.Points[i]
		cx += float64(p.X)
		cy += float64(p.Y)
		cz += float64(p.Z)
		count++
	}
	return cx, cy, cz, count
}

// GetMinMax returns the bounds expanded to include all vertices of the face in float64.
func (fc *Face) GetMinMax(minX, minY, minZ, maxX, maxY, maxZ float64) (float64, float64, float64, float64, float64, float64) {
	for i := range geometry.Triangle3DPoints32 {
		p := &fc.tri.Points[i]
		px, py, pz := float64(p.X), float64(p.Y), float64(p.Z)
		if px < minX {
			minX = px
		}
		if px > maxX {
			maxX = px
		}
		if py < minY {
			minY = py
		}
		if py > maxY {
			maxY = py
		}
		if pz < minZ {
			minZ = pz
		}
		if pz > maxZ {
			maxZ = pz
		}
	}
	return minX, minY, minZ, maxX, maxY, maxZ
}

// Deform applies an impulse near impact within radius, rebuilds the face, and reports whether it changed.
func (fc *Face) Deform(impact geometry.XYZ, impulse geometry.XYZ, radius float64) bool {
	if radius <= 0 || math.IsNaN(radius) || math.IsInf(radius, 0) || !fc.IsDeformable() {
		return false
	}
	if impulse.X == 0 && impulse.Y == 0 && impulse.Z == 0 {
		return false
	}

	// displaceVertex moves the vertex at index by impulse scaled by factor.
	displaceVertex := func(index int, impulse geometry.XYZ, factor float64) {
		p := &fc.tri.Points[index]
		p.X = float32(float64(p.X) + impulse.X*factor)
		p.Y = float32(float64(p.Y) + impulse.Y*factor)
		p.Z = float32(float64(p.Z) + impulse.Z*factor)
	}

	rSq := radius * radius
	deformed := false
	// Local vertex deformation using smoothstep falloff.
	for i := 0; i < geometry.Triangle3DPoints32; i++ {
		p := &fc.tri.Points[i]
		dx := float64(p.X) - impact.X
		dy := float64(p.Y) - impact.Y
		dz := float64(p.Z) - impact.Z
		distSq := dx*dx + dy*dy + dz*dz
		if distSq >= rSq {
			continue
		}
		dist := math.Sqrt(distSq)
		t := 1.0 - dist/radius
		factor := t * t * (3.0 - 2.0*t)
		displaceVertex(i, impulse, factor)
		deformed = true
	}
	// Fallback for large triangles whose vertices are outside
	// the influence radius but whose surface contains the impact.
	if !deformed {
		w0, w1, w2, inside := fc.Barycentric(impact.X, impact.Y, impact.Z)
		if inside {
			weights := [3]float64{w0 * w0, w1 * w1, w2 * w2}
			for i, weight := range weights {
				if weight <= 0 {
					continue
				}
				displaceVertex(i, impulse, weight)
				deformed = true
			}
		}
	}

	if !deformed {
		return false
	}

	fc.Rebuild()

	if fc.parent != nil {
		fc.parent.SetDirty(true)
	}

	return true
}

// DeformDent dents the face inward along its normal at impact by depth within radius, reporting whether it changed.
func (fc *Face) DeformDent(impact geometry.XYZ, depth, radius float64) bool {
	impulse := geometry.XYZ{X: -float64(fc.n.X) * depth, Y: -float64(fc.n.Y) * depth, Z: -float64(fc.n.Z) * depth}
	return fc.Deform(impact, impulse, radius)
}

// NearPoint reports whether p lies within radius of the face's axis-aligned bounding box.
func (fc *Face) NearPoint(p geometry.XYZ, radius float64) bool {
	minX, minY, minZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
	maxX, maxY, maxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64
	minX, minY, minZ, maxX, maxY, maxZ = fc.GetMinMax(minX, minY, minZ, maxX, maxY, maxZ)
	dx, dy, dz := 0.0, 0.0, 0.0
	if p.X < minX {
		dx = minX - p.X
	} else if p.X > maxX {
		dx = p.X - maxX
	}
	if p.Y < minY {
		dy = minY - p.Y
	} else if p.Y > maxY {
		dy = p.Y - maxY
	}
	if p.Z < minZ {
		dz = minZ - p.Z
	} else if p.Z > maxZ {
		dz = p.Z - maxZ
	}
	return dx*dx+dy*dy+dz*dz <= radius*radius
}

// SplitFace subdivides the face according to edge mask bits and given float32 midpoints, preserving interpolated UVs.
func (fc *Face) SplitFace(mask uint8, mab32, mbc32, mca32 geometry.XYZ32) []*Face {
	if mask == 0 {
		return []*Face{fc}
	}
	averageUV := func(a, b float32) float64 {
		return (float64(a) + float64(b)) * 0.5
	}
	a := geometry.XYZ{X: float64(fc.tri.Points[0].X), Y: float64(fc.tri.Points[0].Y), Z: float64(fc.tri.Points[0].Z)}
	b := geometry.XYZ{X: float64(fc.tri.Points[1].X), Y: float64(fc.tri.Points[1].Y), Z: float64(fc.tri.Points[1].Z)}
	c := geometry.XYZ{X: float64(fc.tri.Points[2].X), Y: float64(fc.tri.Points[2].Y), Z: float64(fc.tri.Points[2].Z)}
	mab := geometry.XYZ{X: float64(mab32.X), Y: float64(mab32.Y), Z: float64(mab32.Z)}
	mbc := geometry.XYZ{X: float64(mbc32.X), Y: float64(mbc32.Y), Z: float64(mbc32.Z)}
	mca := geometry.XYZ{X: float64(mca32.X), Y: float64(mca32.Y), Z: float64(mca32.Z)}

	u, v := fc.u, fc.v

	uvAB := [2]float64{averageUV(u[0], u[1]), averageUV(v[0], v[1])}
	uvBC := [2]float64{averageUV(u[1], u[2]), averageUV(v[1], v[2])}
	uvCA := [2]float64{averageUV(u[2], u[0]), averageUV(v[2], v[0])}
	u64 := [3]float64{float64(u[0]), float64(u[1]), float64(u[2])}
	v64 := [3]float64{float64(v[0]), float64(v[1]), float64(v[2])}
	newFace := fc.NewFaceSubdivided

	switch mask {
	case 1:
		return []*Face{
			newFace([3]geometry.XYZ{a, mab, c}, [3]float64{u64[0], uvAB[0], u64[2]}, [3]float64{v64[0], uvAB[1], v64[2]}),
			newFace([3]geometry.XYZ{mab, b, c}, [3]float64{uvAB[0], u64[1], u64[2]}, [3]float64{uvAB[1], v64[1], v64[2]}),
		}
	case 2:
		return []*Face{
			newFace([3]geometry.XYZ{a, b, mbc}, [3]float64{u64[0], u64[1], uvBC[0]}, [3]float64{v64[0], v64[1], uvBC[1]}),
			newFace([3]geometry.XYZ{a, mbc, c}, [3]float64{u64[0], uvBC[0], u64[2]}, [3]float64{v64[0], uvBC[1], v64[2]}),
		}
	case 4:
		return []*Face{
			newFace([3]geometry.XYZ{a, b, mca}, [3]float64{u64[0], u64[1], uvCA[0]}, [3]float64{v64[0], v64[1], uvCA[1]}),
			newFace([3]geometry.XYZ{mca, b, c}, [3]float64{uvCA[0], u64[1], u64[2]}, [3]float64{uvCA[1], v64[1], v64[2]}),
		}
	case 3:
		return []*Face{
			newFace([3]geometry.XYZ{b, mbc, mab}, [3]float64{u64[1], uvBC[0], uvAB[0]}, [3]float64{v64[1], uvBC[1], uvAB[1]}),
			newFace([3]geometry.XYZ{a, mab, c}, [3]float64{u64[0], uvAB[0], u64[2]}, [3]float64{v64[0], uvAB[1], v64[2]}),
			newFace([3]geometry.XYZ{mab, mbc, c}, [3]float64{uvAB[0], uvBC[0], u64[2]}, [3]float64{uvAB[1], uvBC[1], v64[2]}),
		}
	case 5:
		return []*Face{
			newFace([3]geometry.XYZ{a, mab, mca}, [3]float64{u64[0], uvAB[0], uvCA[0]}, [3]float64{v64[0], uvAB[1], uvCA[1]}),
			newFace([3]geometry.XYZ{mab, b, c}, [3]float64{uvAB[0], u64[1], u64[2]}, [3]float64{uvAB[1], v64[1], v64[2]}),
			newFace([3]geometry.XYZ{mab, c, mca}, [3]float64{uvAB[0], u64[2], uvCA[0]}, [3]float64{uvAB[1], v64[2], uvCA[1]}),
		}
	case 6:
		return []*Face{
			newFace([3]geometry.XYZ{c, mca, mbc}, [3]float64{u64[2], uvCA[0], uvBC[0]}, [3]float64{v64[2], uvCA[1], uvBC[1]}),
			newFace([3]geometry.XYZ{a, b, mbc}, [3]float64{u64[0], u64[1], uvBC[0]}, [3]float64{v64[0], v64[1], uvBC[1]}),
			newFace([3]geometry.XYZ{a, mbc, mca}, [3]float64{u64[0], uvBC[0], uvCA[0]}, [3]float64{v64[0], uvBC[1], uvCA[1]}),
		}
	default: // mask == 7
		return []*Face{
			newFace([3]geometry.XYZ{a, mab, mca}, [3]float64{u64[0], uvAB[0], uvCA[0]}, [3]float64{v64[0], uvAB[1], uvCA[1]}),
			newFace([3]geometry.XYZ{mab, b, mbc}, [3]float64{uvAB[0], u64[1], uvBC[0]}, [3]float64{uvAB[1], v64[1], uvBC[1]}),
			newFace([3]geometry.XYZ{mca, mbc, c}, [3]float64{uvCA[0], uvBC[0], u64[2]}, [3]float64{uvCA[1], uvBC[1], v64[2]}),
			newFace([3]geometry.XYZ{mab, mbc, mca}, [3]float64{uvAB[0], uvBC[0], uvCA[0]}, [3]float64{uvAB[1], uvBC[1], uvCA[1]}),
		}
	}
}

// computeNormal calculates and assigns the normal vector (XYZ32) for the Face using float64 intermediate precision.
func (fc *Face) computeNormal() {
	const snapToZero = 1e-15
	const areaCulling = 1e-12

	fc.n = geometry.XYZ32{X: 0, Y: 0, Z: 1}
	p0x, p0y, p0z := fc.Get0()
	p1x, p1y, p1z := fc.Get1()
	p2x, p2y, p2z := fc.Get2()
	v1x, v1y, v1z := p1x-p0x, p1y-p0y, p1z-p0z
	v2x, v2y, v2z := p2x-p0x, p2y-p0y, p2z-p0z
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
		return
	}
	inv := 1.0 / l
	fnx, fny, fnz := float32(nx*inv), float32(ny*inv), float32(nz*inv)
	fc.n.X, fc.n.Y, fc.n.Z = fnx, fny, fnz
	fc.nAbs.X, fc.nAbs.Y, fc.nAbs.Z = float32(math.Abs(float64(fnx))), float32(math.Abs(float64(fny))), float32(math.Abs(float64(fnz)))
}

// computeAABB calculates the axis-aligned bounding box (AABB) for the Face in float64 using an ULP-safe margin.
func (fc *Face) computeAABB() {
	min32X, min32Y, min32Z, max32X, max32Y, max32Z := fc.tri.MinMax()
	minX, minY, minZ := float64(min32X), float64(min32Y), float64(min32Z)
	maxX, maxY, maxZ := float64(max32X), float64(max32Y), float64(max32Z)
	eps := 0.001
	maxAbs := math.Max(math.Max(math.Abs(minX), math.Abs(maxX)), math.Max(math.Max(math.Abs(minY), math.Abs(maxY)), math.Max(math.Abs(minZ), math.Abs(maxZ))))
	if ulpMargin := maxAbs * 1e-6; ulpMargin > eps {
		eps = ulpMargin
	}
	if minZ == maxZ {
		minZ -= eps
		maxZ += eps
	}
	fc.aabb.Rebuild(minX-eps, minY-eps, minZ, maxX+eps, maxY+eps, maxZ)
}

// computeUV computes the UV mapping for the current face based on its normal, material, and texture dimensions.
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
	p0x, p0y, p0z := fc.Get0()
	p1x, p1y, p1z := fc.Get1()
	p2x, p2y, p2z := fc.Get2()
	// Pure Triplanar Projection.
	if nAbsZ >= nAbsX && nAbsZ >= nAbsY {
		// Upper / Lower (Floors and Ceilings)
		fc.SetUV(p0x/w, p0y/h, p1x/w, p1y/h, p2x/w, p2y/h)
	} else if nAbsY >= nAbsX && nAbsY >= nAbsZ {
		// Walls facing Y
		fc.SetUV(p0x/w, p0z/h, p1x/w, p1z/h, p2x/w, p2z/h)
	} else {
		// Walls facing X
		fc.SetUV(p0y/w, p0z/h, p1y/w, p1z/h, p2y/w, p2z/h)
	}
}

// computeTangent derives orthonormal vertex tangents and bitangent handedness in GL UV space (u, -v)
// using float64 intermediate arithmetic and storing compact float32 presentation attributes.
func (fc *Face) computeTangent() {
	p0x, p0y, p0z := fc.Get0()
	p1x, p1y, p1z := fc.Get1()
	p2x, p2y, p2z := fc.Get2()
	e1x, e1y, e1z := p1x-p0x, p1y-p0y, p1z-p0z
	e2x, e2y, e2z := p2x-p0x, p2y-p0y, p2z-p0z

	// GL TexCoords pass V_gl = -v
	du1 := float64(fc.u[1]) - float64(fc.u[0])
	dv1 := float64(fc.v[0]) - float64(fc.v[1])
	du2 := float64(fc.u[2]) - float64(fc.u[0])
	dv2 := float64(fc.v[0]) - float64(fc.v[2])

	det := du1*dv2 - du2*dv1
	var faceTX, faceTY, faceTZ float64
	var faceBX, faceBY, faceBZ float64
	hasUVBasis := math.Abs(det) > 1e-12
	if hasUVBasis {
		invDet := 1.0 / det
		tx := (e1x*dv2 - e2x*dv1) * invDet
		ty := (e1y*dv2 - e2y*dv1) * invDet
		tz := (e1z*dv2 - e2z*dv1) * invDet
		bx := (e2x*du1 - e1x*du2) * invDet
		by := (e2y*du1 - e1y*du2) * invDet
		bz := (e2z*du1 - e1z*du2) * invDet
		tl := math.Sqrt(tx*tx + ty*ty + tz*tz)
		bl := math.Sqrt(bx*bx + by*by + bz*bz)
		if tl > 1e-12 && bl > 1e-12 {
			faceTX, faceTY, faceTZ = tx/tl, ty/tl, tz/tl
			faceBX, faceBY, faceBZ = bx/bl, by/bl, bz/bl
		} else {
			hasUVBasis = false
		}
	}

	if !hasUVBasis {
		ftx, fty, ftz := FallbackTangent(fc.n.X, fc.n.Y, fc.n.Z)
		faceTX, faceTY, faceTZ = float64(ftx), float64(fty), float64(ftz)
		fnx, fny, fnz := float64(fc.n.X), float64(fc.n.Y), float64(fc.n.Z)
		faceBX, faceBY, faceBZ = fny*faceTZ-fnz*faceTY, fnz*faceTX-fnx*faceTZ, fnx*faceTY-fny*faceTX
	}

	for k := 0; k < 3; k++ {
		n32 := fc.vertexNormals[k]
		if n32.X == 0 && n32.Y == 0 && n32.Z == 0 {
			n32 = fc.n
		}
		if n32.X == 0 && n32.Y == 0 && n32.Z == 0 {
			n32 = geometry.XYZ32{X: 0, Y: 0, Z: 1}
		}
		nx, ny, nz := float64(n32.X), float64(n32.Y), float64(n32.Z)
		// Gram-Schmidt orthogonalization: T' = normalize(T - N * dot(N, T))
		dotNT := nx*faceTX + ny*faceTY + nz*faceTZ
		ox, oy, oz := faceTX-nx*dotNT, faceTY-ny*dotNT, faceTZ-nz*dotNT
		ol := math.Sqrt(ox*ox + oy*oy + oz*oz)
		var tox, toy, toz float64
		if ol > 1e-8 {
			invO := 1.0 / ol
			tox, toy, toz = ox*invO, oy*invO, oz*invO
		} else {
			ftx, fty, ftz := FallbackTangent(n32.X, n32.Y, n32.Z)
			tox, toy, toz = float64(ftx), float64(fty), float64(ftz)
		}
		// Bitangent handedness sign = sign(dot(cross(N, T'), B))
		cx := ny*toz - nz*toy
		cy := nz*tox - nx*toz
		cz := nx*toy - ny*tox
		sign := float32(1.0)
		if cx*faceBX+cy*faceBY+cz*faceBZ < 0 {
			sign = -1.0
		}
		fc.tangents[k] = geometry.XYZ32{X: float32(tox), Y: float32(toy), Z: float32(toz)}
		fc.tangentSigns[k] = sign
	}
}
