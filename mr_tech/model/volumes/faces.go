package volumes

import (
	"fmt"
	"math"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
	"github.com/markel1974/godoom/mr_tech/textures"
)

type edgeFaceRef struct {
	faceIdx int32
	edgeIdx uint8
}

type edgeAdjacency struct {
	refs  [2]edgeFaceRef
	count uint8
}

type adjFace struct {
	idx int32
	nx  float64
	ny  float64
	nz  float64
	ax  float64
	ay  float64
	az  float64
}

type smoothVertKey struct {
	x, y, z int64
}

func makeSmoothVertKey(p geometry.XYZ32) smoothVertKey {
	return smoothVertKey{
		x: int64(math.Round(float64(p.X) * 100.0)),
		y: int64(math.Round(float64(p.Y) * 100.0)),
		z: int64(math.Round(float64(p.Z) * 100.0)),
	}
}

// Faces stores a collection of Face values and accumulated geometric properties.
type Faces struct {
	tree             *aabb.Tree
	container        []*Face
	count            int
	cx, cy, cz       float64
	ixx, ixy, ixz    float64
	iyy, iyz, izz    float64
	minX, minY, minZ float64
	maxX, maxY, maxZ float64
	w, h, d          float64

	midpoints       map[EdgeKey]geometry.XYZ32
	displacements   map[VertexKey]geometry.XYZ32
	edgeAdj         map[EdgeKey]edgeAdjacency
	smoothHead      map[smoothVertKey]int32
	smoothNext      []int32
	adjFaceData     []adjFace
	nextFaces       []*Face
	discarded       []*Face
	hasTesselation  bool
	allowTessellate bool
	version         uint64
	geomHash        uint64
	animMats        []*textures.Material
	renderHash      uint64
	renderHashVer   uint64
	renderHashReady bool
}

// NewFaces creates a Faces collection sized to the given tree capacity.
// If fill is true, the collection length is initialized to initSize.
func NewFaces(tree *aabb.Tree, fill bool) *Faces {
	initSize := 0
	if tree != nil {
		initSize = int(tree.Capacity())
	}
	return NewFacesSized(tree, initSize, fill)
}

// NewFacesSized creates a Faces collection with an explicit initial slice capacity and optional tree.
// If fill is true, the collection length is initialized to initSize.
func NewFacesSized(tree *aabb.Tree, initSize int, fill bool) *Faces {
	if initSize < 0 {
		initSize = 0
	}
	count := 0
	if fill {
		count = initSize
	}
	return &Faces{
		tree:            tree,
		container:       make([]*Face, initSize),
		count:           count,
		midpoints:       nil,
		displacements:   nil,
		nextFaces:       nil,
		hasTesselation:  false,
		allowTessellate: true,
	}
}

// SetAllowTessellate enables or disables dynamic tessellation (subdivision) during deformation.
func (f *Faces) SetAllowTessellate(allow bool) {
	f.allowTessellate = allow
}

// IsAllowTessellate reports whether dynamic tessellation is allowed during deformation.
func (f *Faces) IsAllowTessellate() bool {
	return f.allowTessellate
}

func (f *Faces) TessellateInit() {
	if f.hasTesselation {
		return
	}
	f.midpoints = make(map[EdgeKey]geometry.XYZ32, 128)
	f.displacements = make(map[VertexKey]geometry.XYZ32, 128)
	f.edgeAdj = make(map[EdgeKey]edgeAdjacency, max(f.count*2, 128))
	f.smoothHead = make(map[smoothVertKey]int32, max(f.count*2, 128))
	f.nextFaces = make([]*Face, 0, f.count)
	f.hasTesselation = true
}

// GetVersion returns a monotonically increasing revision counter incremented when faces change.
func (f *Faces) GetVersion() uint64 {
	return f.version
}

// BumpVersion increments the faces revision counter to signal geometry updates.
func (f *Faces) BumpVersion() {
	f.version++
}

// ApplyCachedSummary sets precomputed face count, bounding extents, centroid, inertia tensor, and geometry hash,
// and increments the version counter.
func (f *Faces) ApplyCachedSummary(count int, minX, minY, minZ, maxX, maxY, maxZ, w, h, d, cx, cy, cz, ixx, ixy, ixz, iyy, iyz, izz float64, geomHash uint64) {
	f.count = count
	f.minX, f.minY, f.minZ = minX, minY, minZ
	f.maxX, f.maxY, f.maxZ = maxX, maxY, maxZ
	f.w, f.h, f.d = w, h, d
	f.cx, f.cy, f.cz = cx, cy, cz
	f.ixx, f.ixy, f.ixz = ixx, ixy, ixz
	f.iyy, f.iyz, f.izz = iyy, iyz, izz
	f.geomHash = geomHash
	f.version++
}

// GetGeomHash returns the structural and vertex fingerprint for sharing identical static GPU meshes.
func (f *Faces) GetGeomHash() uint64 {
	return f.geomHash
}

// GetRenderHash returns a content hash covering geometry revision, face identity, and animated material frames.
func (f *Faces) GetRenderHash() uint64 {
	if f == nil || f.count == 0 {
		return 0
	}
	if !f.renderHashReady || f.renderHashVer != f.version {
		h := geometry.MixUintptr(uintptr(unsafe.Pointer(f)))
		h = geometry.MixU64(h, f.version)
		h = geometry.MixU64(h, f.geomHash)
		h = geometry.MixU64(h, uint64(f.count))
		if h == 0 {
			h = 1
		}
		f.renderHash = h
		f.animMats = f.animMats[:0]
		for i := 0; i < f.count; i++ {
			fc := f.container[i]
			if fc == nil || fc.material == nil || !fc.material.IsAnimated() {
				continue
			}
			dup := false
			for _, m := range f.animMats {
				if m == fc.material {
					dup = true
					break
				}
			}
			if !dup {
				f.animMats = append(f.animMats, fc.material)
			}
		}
		f.renderHashVer = f.version
		f.renderHashReady = true
	}
	h := f.renderHash
	for _, mat := range f.animMats {
		h = geometry.MixU64(h, geometry.MixUintptr(uintptr(unsafe.Pointer(mat.GetCurrentFrame()))))
	}
	if h == 0 {
		return 1
	}
	return h
}

// Rebuild recomputes cached bounds, normals, active edge masks, mass properties, and the spatial tree.
func (f *Faces) Rebuild() {
	f.RebuildGeometry()
	f.RebuildTree()
}

// RebuildGeometry recomputes cached bounds, normals, tangents, geometry hash, active edges, and mass properties
// without modifying the spatial AABB tree.
func (f *Faces) RebuildGeometry() {
	f.version++
	if f.count == 0 {
		f.minX, f.minY, f.minZ, f.maxX, f.maxY, f.maxZ = 0, 0, 0, 0, 0, 0
		f.w, f.h, f.d = 0, 0, 0
		f.cx, f.cy, f.cz = 0, 0, 0
		f.ixx, f.ixy, f.ixz, f.iyy, f.iyz, f.izz = 0, 0, 0, 0, 0, 0
		f.geomHash = 0
		return
	}
	f.minX, f.minY, f.minZ = math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
	f.maxX, f.maxY, f.maxZ = -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64
	for x := 0; x < f.count; x++ {
		face := f.container[x]
		face.rebuildBase()
		f.minX, f.minY, f.minZ, f.maxX, f.maxY, f.maxZ = face.GetMinMax(f.minX, f.minY, f.minZ, f.maxX, f.maxY, f.maxZ)
	}
	f.computeActiveEdges()
	f.computeVertexNormalsAndTangents(0.5)
	f.computeGeomHash()
	f.cx, f.cy, f.cz, f.ixx, f.ixy, f.ixz, f.iyy, f.iyz, f.izz = f.ComputeMassProperties()
	f.w = f.maxX - f.minX
	f.h = f.maxY - f.minY
	f.d = f.maxZ - f.minZ
}

// RebuildTree clears and repopulates the associated spatial AABB tree from the current faces
// without recomputing geometry properties or bumping the version counter.
func (f *Faces) RebuildTree() {
	if f.tree == nil {
		return
	}
	f.discarded = f.discarded[:0]
	if f.count > 0 {
		f.tree.EnsureCapacity(uint(f.count * 2))
	}
	f.tree.Clear()
	for x := 0; x < f.count; x++ {
		if face := f.container[x]; face != nil {
			f.tree.InsertObject(face)
		}
	}
}

// SyncDeformedTree incrementally removes replaced split faces, updates modified leaves,
// and inserts newly subdivided faces into the owned spatial AABB tree.
func (f *Faces) SyncDeformedTree() {
	if f.tree == nil {
		return
	}
	if f.tree.IsEmpty() {
		f.RebuildTree()
		return
	}
	if f.count > 0 {
		f.tree.EnsureCapacity(uint(f.count * 2))
	}
	for _, oldFace := range f.discarded {
		f.tree.RemoveObject(oldFace)
	}
	f.discarded = f.discarded[:0]
	for x := 0; x < f.count; x++ {
		face := f.container[x]
		if face == nil {
			continue
		}
		if f.tree.ContainsObject(face) {
			f.tree.UpdateObject(face)
		} else {
			f.tree.InsertObject(face)
		}
	}
}

// computeVertexNormalsAndTangents averages area-weighted face normals at shared vertices
// when adjacent faces fall within the crease threshold (cosCrease = 0.5 -> ~60 deg),
// while preserving explicit artist vertex normals when already set.
func (f *Faces) computeVertexNormalsAndTangents(cosCrease float64) {
	if f.count <= 0 {
		return
	}
	needsSmooth := false
	if f.count > 1 {
		for i := 0; i < f.count; i++ {
			if fc := f.container[i]; fc != nil && !fc.hasExplicitNormals {
				needsSmooth = true
				break
			}
		}
	}
	if !needsSmooth {
		for i := 0; i < f.count; i++ {
			if fc := f.container[i]; fc != nil {
				fc.computeTangent()
			}
		}
		return
	}

	neededLinks := f.count * 3
	var smoothHead map[smoothVertKey]int32
	var smoothNext []int32
	var faceData []adjFace
	if f.hasTesselation {
		if f.smoothHead == nil {
			f.smoothHead = make(map[smoothVertKey]int32, f.count*2)
		} else {
			clear(f.smoothHead)
		}
		if cap(f.smoothNext) < neededLinks {
			f.smoothNext = make([]int32, neededLinks)
		}
		if cap(f.adjFaceData) < f.count {
			f.adjFaceData = make([]adjFace, f.count)
		}
		smoothHead = f.smoothHead
		smoothNext = f.smoothNext[:neededLinks]
		faceData = f.adjFaceData[:f.count]
	} else {
		smoothHead = make(map[smoothVertKey]int32, f.count*2)
		smoothNext = make([]int32, neededLinks)
		faceData = make([]adjFace, f.count)
	}

	for i := 0; i < f.count; i++ {
		fc := f.container[i]
		if fc == nil {
			continue
		}
		p0x, p0y, p0z := fc.Get0()
		p1x, p1y, p1z := fc.Get1()
		p2x, p2y, p2z := fc.Get2()
		e1x, e1y, e1z := p1x-p0x, p1y-p0y, p1z-p0z
		e2x, e2y, e2z := p2x-p0x, p2y-p0y, p2z-p0z
		ax := e1y*e2z - e1z*e2y
		ay := e1z*e2x - e1x*e2z
		az := e1x*e2y - e1y*e2x
		faceData[i] = adjFace{
			idx: int32(i),
			nx:  float64(fc.n.X), ny: float64(fc.n.Y), nz: float64(fc.n.Z),
			ax: ax, ay: ay, az: az,
		}
		for k := range 3 {
			key := makeSmoothVertKey(fc.tri.Points[k])
			slot := i*3 + k
			smoothNext[slot] = smoothHead[key]
			smoothHead[key] = int32(slot + 1)
		}
	}

	for i := 0; i < f.count; i++ {
		fc := f.container[i]
		if fc == nil {
			continue
		}
		if fc.hasExplicitNormals {
			fc.computeTangent()
			continue
		}
		fd := faceData[i]
		if fd.nx == 0 && fd.ny == 0 && fd.nz == 0 {
			fc.computeTangent()
			continue
		}
		for k := range 3 {
			key := makeSmoothVertKey(fc.tri.Points[k])
			var sx, sy, sz float64
			for cur := smoothHead[key]; cur > 0; cur = smoothNext[cur-1] {
				ofd := faceData[(cur-1)/3]
				if fd.nx*ofd.nx+fd.ny*ofd.ny+fd.nz*ofd.nz >= cosCrease {
					sx += ofd.ax
					sy += ofd.ay
					sz += ofd.az
				}
			}
			ptr := &fc.vertexNormals[k]
			ptr.X, ptr.Y, ptr.Z = NormalizeXYZ64(sx, sy, sz, fc.n.X, fc.n.Y, fc.n.Z)
		}
		fc.computeTangent()
	}
}

func (f *Faces) computeGeomHash() {
	const offset64 uint64 = 14695981039346656037
	const prime64 uint64 = 1099511628211
	h := offset64
	mixU32 := func(v uint32) {
		h ^= uint64(v)
		h *= prime64
	}
	mixF32 := func(v float32) {
		mixU32(math.Float32bits(v))
	}
	mixStr := func(s string) {
		for i := 0; i < len(s); i++ {
			h ^= uint64(s[i])
			h *= prime64
		}
		h ^= 0xff
		h *= prime64
	}

	mixU32(uint32(f.count))
	for i := 0; i < f.count; i++ {
		fc := f.container[i]
		if fc == nil {
			continue
		}
		for k := 0; k < 3; k++ {
			p := fc.tri.Points[k]
			mixF32(p.X)
			mixF32(p.Y)
			mixF32(p.Z)
			mixF32(fc.u[k])
			mixF32(fc.v[k])
			mixF32(fc.vertexNormals[k].X)
			mixF32(fc.vertexNormals[k].Y)
			mixF32(fc.vertexNormals[k].Z)
		}
		mixU32(uint32(fc.lightmapID))
		if fc.material != nil {
			mixStr(fc.material.GetName())
		}
	}
	f.geomHash = h
}

// computeActiveEdges classifies shared triangle edges within f as active (convex/boundary) or inactive (coplanar/concave)
// using a short-lived local adjacency map so no scratch buckets remain pinned after computation.
func (f *Faces) computeActiveEdges() {
	const (
		coplanarCosThreshold = 0.9995
		coplanarSinMax       = 0.032
		concaveSinMin        = 1e-4
	)
	for i := 0; i < f.count; i++ {
		if fc := f.container[i]; fc != nil {
			fc.activeEdges = EdgeAllActive
		}
	}
	if f.count < 2 {
		return
	}
	var edgeAdj map[EdgeKey]edgeAdjacency
	if f.hasTesselation {
		if f.edgeAdj == nil {
			f.edgeAdj = make(map[EdgeKey]edgeAdjacency, f.count*2)
		} else {
			clear(f.edgeAdj)
		}
		edgeAdj = f.edgeAdj
	} else {
		edgeAdj = make(map[EdgeKey]edgeAdjacency, f.count*2)
	}

	canOccludeEdge := func(fc *Face) bool {
		if fc == nil {
			return false
		}
		if fc.n.X == 0 && fc.n.Y == 0 && fc.n.Z == 0 {
			return false
		}
		// Keep untextured collision/test faces enabled alongside textured solid faces; ignore liquid/sky overlays.
		return fc.material == nil || fc.IsSolid()
	}

	for i := 0; i < f.count; i++ {
		fc := f.container[i]
		if !canOccludeEdge(fc) {
			continue
		}
		p := fc.tri.Points
		keys := [3]EdgeKey{MakeEdgeKey(p[0], p[1]), MakeEdgeKey(p[1], p[2]), MakeEdgeKey(p[2], p[0])}
		for eIdx := uint8(0); eIdx < 3; eIdx++ {
			key := keys[eIdx]
			adj := edgeAdj[key]
			if adj.count < 2 {
				adj.refs[adj.count] = edgeFaceRef{faceIdx: int32(i), edgeIdx: eIdx}
			}
			if adj.count < 255 {
				adj.count++
			}
			edgeAdj[key] = adj
		}
	}

	sinToPlane := func(opp, anchor, n geometry.XYZ32) float64 {
		dx := float64(opp.X) - float64(anchor.X)
		dy := float64(opp.Y) - float64(anchor.Y)
		dz := float64(opp.Z) - float64(anchor.Z)
		lenSq := dx*dx + dy*dy + dz*dz
		if lenSq <= 1e-18 {
			return 0.0
		}
		return (dx*float64(n.X) + dy*float64(n.Y) + dz*float64(n.Z)) / math.Sqrt(lenSq)
	}

	for _, adj := range edgeAdj {
		if adj.count != 2 {
			continue
		}
		refA := adj.refs[0]
		refB := adj.refs[1]
		fa := f.container[refA.faceIdx]
		fb := f.container[refB.faceIdx]
		if fa == fb {
			continue
		}
		if (fa.material == nil) != (fb.material == nil) {
			continue
		}
		pA, _, oppA := fa.GetEdgePoints(int(refA.edgeIdx))
		_, _, oppB := fb.GetEdgePoints(int(refB.edgeIdx))

		cosAngle := float64(fa.n.X)*float64(fb.n.X) + float64(fa.n.Y)*float64(fb.n.Y) + float64(fa.n.Z)*float64(fb.n.Z)
		sinB := sinToPlane(oppB, pA, fa.n)
		sinA := sinToPlane(oppA, pA, fb.n)

		coplanar := cosAngle >= coplanarCosThreshold && math.Abs(sinB) <= coplanarSinMax && math.Abs(sinA) <= coplanarSinMax
		concave := sinB > concaveSinMin && sinA > concaveSinMin
		if coplanar || concave {
			fa.SetEdgeActive(int(refA.edgeIdx), false)
			fb.SetEdgeActive(int(refB.edgeIdx), false)
		}
	}
}

// GetTree returns the AABB tree containing the faces.
func (f *Faces) GetTree() *aabb.Tree {
	return f.tree
}

// GetMin returns the cached minimum X, Y, and Z coordinates of the faces' bounding box.
func (f *Faces) GetMin() (float64, float64, float64) {
	return f.minX, f.minY, f.minZ
}

// GetMax returns the cached maximum X, Y, and Z coordinates of the faces' bounding box.
func (f *Faces) GetMax() (float64, float64, float64) {
	return f.maxX, f.maxY, f.maxZ
}

// GetMinMax returns the cached minimum and maximum X, Y, and Z coordinates.
func (f *Faces) GetMinMax() (float64, float64, float64, float64, float64, float64) {
	return f.minX, f.minY, f.minZ, f.maxX, f.maxY, f.maxZ
}

// GetSize returns the cached width, height, and depth of the faces' bounding box.
func (f *Faces) GetSize() (float64, float64, float64) {
	return f.w, f.h, f.d
}

// GetInertia returns the stored inertia tensor components for the faces.
func (f *Faces) GetInertia() (ixx, ixy, ixz, iyy, iyz, izz float64) {
	return f.ixx, f.ixy, f.ixz, f.iyy, f.iyz, f.izz
}

// GetCentroid returns the cached centroid coordinates of the faces.
func (f *Faces) GetCentroid() (float64, float64, float64) {
	return f.cx, f.cy, f.cz
}

// Len returns the number of faces in the collection.
func (f *Faces) Len() int {
	return f.count
}

// Add appends a face to the collection, growing the internal storage when needed.
func (f *Faces) Add(face *Face) {
	if f.count >= len(f.container) {
		newCap := f.count * 2
		if newCap == 0 {
			newCap = 1
		}
		newFaces := make([]*Face, newCap)
		copy(newFaces, f.container)
		f.container = newFaces
	}
	f.container[f.count] = face
	f.count++
	f.version++
}

// ComputeMassProperties computes the centroid and unit-mass inertia tensor for the stored faces.
// It returns zero values when there are no faces, no valid face, or the signed volume is zero.
func (f *Faces) ComputeMassProperties() (float64, float64, float64, float64, float64, float64, float64, float64, float64) {
	if f.count == 0 {
		return 0, 0, 0, 0, 0, 0, 0, 0, 0
	}
	// Use a vertex as reference to improve numerical stability.
	refX, refY, refZ := f.container[0].GetPoint0()
	var volume6 float64
	// First moments.
	var mx, my, mz float64
	// Second moments matrix.
	var sxx, sxy, sxz float64
	var syy, syz, szz float64
	for i := 0; i < f.count; i++ {
		face := f.container[i]
		volume6, mx, my, mz, sxx, sxy, sxz, syy, syz, szz = rigid_body.CompileMassAccumulate(refX, refY, refZ, face, volume6, mx, my, mz, sxx, sxy, sxz, syy, syz, szz)
	}
	if volume6 == 0 {
		return 0, 0, 0, 0, 0, 0, 0, 0, 0
	}
	cx, cy, cz, ixx, ixy, ixz, iyy, iyz, izz := rigid_body.CompileMassFinalize(volume6, refX, refY, refZ, mx, my, mz, sxx, sxy, sxz, syy, syz, szz)
	return cx, cy, cz, ixx, ixy, ixz, iyy, iyz, izz
}

// Set assigns face to the element at idx.
func (f *Faces) Set(idx int, face *Face) {
	f.container[idx] = face
	f.version++
}

// At returns the face stored at idx, or nil if idx is out of bounds.
func (f *Faces) At(idx int) *Face {
	if idx < 0 || idx >= len(f.container) {
		return nil
	}
	return f.container[idx]
}

// EnsureLen ensures the Faces container has at least targetLen elements, appending empty faces if needed.
func (f *Faces) EnsureLen(targetLen int) {
	for f.count < targetLen {
		f.Add(NewFaceEmpty(nil, [3]float64{}, [3]float64{}))
	}
}

// SetCount sets the active face count, keeping it within the container length.
func (f *Faces) SetCount(count int) {
	if count <= len(f.container) {
		if f.count != count {
			f.count = count
			f.version++
		}
	}
}

// Clear removes all faces from the collection.
func (f *Faces) Clear() {
	if f.tree != nil {
		f.tree.Clear()
	}
	f.count = 0
	f.version++
}

// Deform applies impulse to deformable vertices within radius of impact.
// Deform returns true when at least one vertex is displaced.
func (f *Faces) Deform(impact geometry.XYZ, impulse geometry.XYZ, radius float64) bool {
	if f.allowTessellate {
		// Quattro intervalli geometrici lungo il diametro d'impatto.
		targetEdge := radius * 0.45
		if targetEdge < deformMinEdge {
			targetEdge = deformMinEdge
		}
		if _, err := f.TessellateNear(impact, radius, targetEdge, deformMaxFaces); err != nil {
			return false
		}
	}
	f.TessellateInit()
	radiusSq := radius * radius

	clear(f.displacements)

	for i := 0; i < f.count; i++ {
		fc := f.container[i]
		if fc == nil || !fc.IsDeformable() || !fc.NearPoint(impact, radius) {
			continue
		}
		for _, p := range fc.tri.Points {
			key := MakeVertexKey(p)
			if _, exists := f.displacements[key]; exists {
				continue
			}
			dx := float64(p.X) - impact.X
			dy := float64(p.Y) - impact.Y
			dz := float64(p.Z) - impact.Z
			distSq := dx*dx + dy*dy + dz*dz
			if distSq >= radiusSq {
				continue
			}
			t := 1.0 - math.Sqrt(distSq)/radius
			factor := t * t * (3.0 - 2.0*t)
			f.displacements[key] = geometry.XYZ32{X: float32(impulse.X * factor), Y: float32(impulse.Y * factor), Z: float32(impulse.Z * factor)}
		}
	}

	// Fallback per triangoli grandi i cui vertici ricadono all'esterno del raggio d'impatto
	usedFallback := false
	if len(f.displacements) == 0 {
		usedFallback = true
		for i := 0; i < f.count; i++ {
			fc := f.container[i]
			if fc == nil || !fc.IsDeformable() {
				continue
			}
			w0, w1, w2, inside := fc.Barycentric(impact.X, impact.Y, impact.Z)
			if !inside {
				continue
			}
			p0x, p0y, p0z := fc.Get0()
			nx, ny, nz := fc.GetNormal()
			distPlane := math.Abs((impact.X-p0x)*nx + (impact.Y-p0y)*ny + (impact.Z-p0z)*nz)
			if distPlane > radius*2.0 && distPlane > 16.0 {
				continue
			}
			weights := [3]float64{w0 * w0, w1 * w1, w2 * w2}
			for j, weight := range weights {
				if weight <= 0 {
					continue
				}
				p := fc.tri.Points[j]
				key := MakeVertexKey(p)
				dx, dy, dz := impulse.X*weight, impulse.Y*weight, impulse.Z*weight
				disp := geometry.XYZ32{X: float32(dx), Y: float32(dy), Z: float32(dz)}
				if existing, ok := f.displacements[key]; ok {
					if dx*dx+dy*dy+dz*dz > float64(existing.X)*float64(existing.X)+float64(existing.Y)*float64(existing.Y)+float64(existing.Z)*float64(existing.Z) {
						f.displacements[key] = disp
					}
				} else {
					f.displacements[key] = disp
				}
			}
		}
	}

	if len(f.displacements) == 0 {
		return false
	}

	// Aggiorna tutte le copie locali dei vertici condivisi.
	for i := 0; i < f.count; i++ {
		fc := f.container[i]
		if fc == nil || (!usedFallback && !fc.NearPoint(impact, radius)) {
			continue
		}
		for j := 0; j < geometry.Triangle3DPoints32; j++ {
			p := &fc.tri.Points[j]
			if displacement, ok := f.displacements[MakeVertexKey(*p)]; ok {
				p.X += displacement.X
				p.Y += displacement.Y
				p.Z += displacement.Z
			}
		}
	}

	f.version++
	return true
}

// TessellateNear raffina localmente la mesh mantenendo la conformità dei lati.
// maxEdge è la lunghezza massima desiderata degli spigoli.
// maxFaces <= 0 disabilita il limite sul numero di facce.
func (f *Faces) TessellateNear(impact geometry.XYZ, radius, maxEdge float64, maxFaces int) (int, error) {
	if f == nil || radius <= 0 || maxEdge <= 0 {
		return 0, fmt.Errorf("invalid tessellation parameters")
	}
	if maxEdge < deformMinEdge {
		maxEdge = deformMinEdge
	}

	midpoint := func(a, b geometry.XYZ32) geometry.XYZ32 {
		return geometry.XYZ32{X: (a.X + b.X) * 0.5, Y: (a.Y + b.Y) * 0.5, Z: (a.Z + b.Z) * 0.5}
	}

	distanceSquared := func(a, b geometry.XYZ32) float64 {
		dx := float64(a.X) - float64(b.X)
		dy := float64(a.Y) - float64(b.Y)
		dz := float64(a.Z) - float64(b.Z)
		return dx*dx + dy*dy + dz*dz
	}

	f.TessellateInit()

	initialCount := f.count
	maxEdgeSq := maxEdge * maxEdge
	searchRadius := radius + maxEdge
	searchRadiusSq := searchRadius * searchRadius
	coarseCut := searchRadius * 1.5
	if coarseCut < maxEdge {
		coarseCut = maxEdge
	}
	coarseCutSq := coarseCut * coarseCut

	for pass := 0; pass < deformMaxSubdivisionPasses; pass++ {
		clear(f.midpoints)
		// individua gli spigoli da raffinare nella regione d'impatto.
		for i := 0; i < f.count; i++ {
			fc := f.container[i]
			if fc == nil || !fc.IsDeformable() {
				continue
			}
			if !fc.NearPoint(impact, searchRadius) {
				continue
			}
			p := fc.tri.Points
			if distPointToTriangleSq(impact, p[0], p[1], p[2]) > searchRadiusSq {
				continue
			}
			edges := [3][2]geometry.XYZ32{{p[0], p[1]}, {p[1], p[2]}, {p[2], p[0]}}
			for _, edge := range edges {
				d2 := distanceSquared(edge[0], edge[1])
				// Non suddividere se lo spigolo è già più corto di deformMinEdge
				if d2 <= maxEdgeSq || d2 <= minEdgeSq {
					continue
				}
				if d2 <= coarseCutSq && distPointToSegmentSq(impact, edge[0], edge[1]) > searchRadiusSq {
					continue
				}
				key := MakeEdgeKey(edge[0], edge[1])
				if _, exists := f.midpoints[key]; !exists {
					f.midpoints[key] = midpoint(edge[0], edge[1])
				}
			}
		}
		if len(f.midpoints) == 0 {
			break
		}
		// applica le suddivisioni a tutte le facce incidenti.
		f.nextFaces = f.nextFaces[:0]
		for i := 0; i < f.count; i++ {
			fc := f.container[i]
			if fc == nil {
				continue
			}
			p := fc.tri.Points
			var mask uint8
			var mab, mbc, mca geometry.XYZ32
			if m, ok := f.midpoints[MakeEdgeKey(p[0], p[1])]; ok {
				mask |= 1
				mab = m
			}
			if m, ok := f.midpoints[MakeEdgeKey(p[1], p[2])]; ok {
				mask |= 2
				mbc = m
			}
			if m, ok := f.midpoints[MakeEdgeKey(p[2], p[0])]; ok {
				mask |= 4
				mca = m
			}
			if mask == 0 {
				f.nextFaces = append(f.nextFaces, fc)
			} else {
				if fc.GetAABB().GetIndex() >= 0 {
					f.discarded = append(f.discarded, fc)
				}
				fc.appendSplitFaces(mask, mab, mbc, mca, &f.nextFaces)
			}
			if maxFaces > 0 && len(f.nextFaces) > maxFaces {
				return f.count - initialCount, fmt.Errorf("tessellation face limit exceeded: %d", maxFaces)
			}
		}
		f.container, f.nextFaces = f.nextFaces, f.container
		f.count = len(f.container)
	}
	return f.count - initialCount, nil
}

// distPointToSegmentSq returns the squared distance from p to the line segment from a to b.
// It treats a near-zero-length segment as a single point at a.
func distPointToSegmentSq(p geometry.XYZ, a, b geometry.XYZ32) float64 {
	abX := float64(b.X) - float64(a.X)
	abY := float64(b.Y) - float64(a.Y)
	abZ := float64(b.Z) - float64(a.Z)
	apX := p.X - float64(a.X)
	apY := p.Y - float64(a.Y)
	apZ := p.Z - float64(a.Z)
	abLenSq := abX*abX + abY*abY + abZ*abZ
	if abLenSq <= 1e-18 {
		return apX*apX + apY*apY + apZ*apZ
	}
	t := (apX*abX + apY*abY + apZ*abZ) / abLenSq
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	cx := apX - abX*t
	cy := apY - abY*t
	cz := apZ - abZ*t
	return cx*cx + cy*cy + cz*cz
}

func distPointToTriangleSq(p geometry.XYZ, a32, b32, c32 geometry.XYZ32) float64 {
	ax, ay, az := float64(a32.X), float64(a32.Y), float64(a32.Z)
	bx, by, bz := float64(b32.X), float64(b32.Y), float64(b32.Z)
	cx, cy, cz := float64(c32.X), float64(c32.Y), float64(c32.Z)

	abX, abY, abZ := bx-ax, by-ay, bz-az
	acX, acY, acZ := cx-ax, cy-ay, cz-az
	apX, apY, apZ := p.X-ax, p.Y-ay, p.Z-az

	d1 := abX*apX + abY*apY + abZ*apZ
	d2 := acX*apX + acY*apY + acZ*apZ
	if d1 <= 0 && d2 <= 0 {
		return apX*apX + apY*apY + apZ*apZ
	}

	bpX, bpY, bpZ := p.X-bx, p.Y-by, p.Z-bz
	d3 := abX*bpX + abY*bpY + abZ*bpZ
	d4 := acX*bpX + acY*bpY + acZ*bpZ
	if d3 >= 0 && d4 <= d3 {
		return bpX*bpX + bpY*bpY + bpZ*bpZ
	}

	vc := d1*d4 - d3*d2
	if vc <= 0 && d1 >= 0 && d3 <= 0 {
		if denom := d1 - d3; denom > 1e-18 {
			v := d1 / denom
			dx, dy, dz := apX-abX*v, apY-abY*v, apZ-abZ*v
			return dx*dx + dy*dy + dz*dz
		}
	}

	cpX, cpY, cpZ := p.X-cx, p.Y-cy, p.Z-cz
	d5 := abX*cpX + abY*cpY + abZ*cpZ
	d6 := acX*cpX + acY*cpY + acZ*cpZ
	if d6 >= 0 && d5 <= d6 {
		return cpX*cpX + cpY*cpY + cpZ*cpZ
	}

	vb := d5*d2 - d1*d6
	if vb <= 0 && d2 >= 0 && d6 <= 0 {
		if denom := d2 - d6; denom > 1e-18 {
			w := d2 / denom
			dx, dy, dz := apX-acX*w, apY-acY*w, apZ-acZ*w
			return dx*dx + dy*dy + dz*dz
		}
	}

	va := d3*d6 - d5*d4
	if va <= 0 && (d4-d3) >= 0 && (d5-d6) >= 0 {
		if denom := (d4 - d3) + (d5 - d6); denom > 1e-18 {
			w := (d4 - d3) / denom
			bcX, bcY, bcZ := cx-bx, cy-by, cz-bz
			dx, dy, dz := bpX-bcX*w, bpY-bcY*w, bpZ-bcZ*w
			return dx*dx + dy*dy + dz*dz
		}
	}

	denom := va + vb + vc
	if denom <= 1e-18 {
		dAB := distPointToSegmentSq(p, a32, b32)
		dBC := distPointToSegmentSq(p, b32, c32)
		dCA := distPointToSegmentSq(p, c32, a32)
		if dBC < dAB {
			dAB = dBC
		}
		if dCA < dAB {
			dAB = dCA
		}
		return dAB
	}
	inv := 1.0 / denom
	v := vb * inv
	w := vc * inv
	dx := apX - abX*v - acX*w
	dy := apY - abY*v - acY*w
	dz := apZ - abZ*v - acZ*w
	return dx*dx + dy*dy + dz*dz
}
