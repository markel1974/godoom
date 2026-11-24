package volumes

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

type edgeFaceRef struct {
	faceIdx int32
	edgeIdx uint8
}

type edgeAdjacency struct {
	refs  [2]edgeFaceRef
	count uint8
}

type smoothVertKey struct {
	x, y, z int64
}

func makeSmoothVertKey(p geometry.XYZ) smoothVertKey {
	return smoothVertKey{
		x: int64(math.Round(p.X * 100.0)),
		y: int64(math.Round(p.Y * 100.0)),
		z: int64(math.Round(p.Z * 100.0)),
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

	markedEdges     map[EdgeKey]struct{}
	midpoints       map[EdgeKey]geometry.XYZ
	displacements   map[VertexKey]geometry.XYZ
	nextFaces       []*Face
	hasTesselation  bool
	allowTessellate bool
	version         uint64
	geomHash        uint64
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
		markedEdges:     nil,
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
	f.markedEdges = make(map[EdgeKey]struct{}, 128)
	f.midpoints = make(map[EdgeKey]geometry.XYZ, 128)
	f.displacements = make(map[VertexKey]geometry.XYZ, 128)
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
		face.Rebuild()
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
	f.tree.Clear()
	for x := 0; x < f.count; x++ {
		if face := f.container[x]; face != nil {
			f.tree.InsertObject(face)
		}
	}
}

// computeVertexNormalsAndTangents averages area-weighted face normals at shared vertices
// when adjacent faces fall within the crease threshold (cosCrease = 0.5 -> ~60 deg),
// while preserving explicit artist vertex normals when already set.
func (f *Faces) computeVertexNormalsAndTangents(cosCrease float64) {
	if f.count <= 1 {
		return
	}
	needsSmooth := false
	for i := 0; i < f.count; i++ {
		if fc := f.container[i]; fc != nil && !fc.hasExplicitNormals {
			needsSmooth = true
			break
		}
	}
	if !needsSmooth {
		return
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
	adjMap := make(map[smoothVertKey][]int32, f.count*2)
	faceData := make([]adjFace, f.count)

	for i := 0; i < f.count; i++ {
		fc := f.container[i]
		if fc == nil {
			continue
		}
		p0, p1, p2 := fc.tri.Points[0], fc.tri.Points[1], fc.tri.Points[2]
		e1x, e1y, e1z := p1.X-p0.X, p1.Y-p0.Y, p1.Z-p0.Z
		e2x, e2y, e2z := p2.X-p0.X, p2.Y-p0.Y, p2.Z-p0.Z
		ax := e1y*e2z - e1z*e2y
		ay := e1z*e2x - e1x*e2z
		az := e1x*e2y - e1y*e2x
		faceData[i] = adjFace{
			idx: int32(i),
			nx:  fc.n.X, ny: fc.n.Y, nz: fc.n.Z,
			ax: ax, ay: ay, az: az,
		}
		for k := 0; k < 3; k++ {
			key := makeSmoothVertKey(fc.tri.Points[k])
			adjMap[key] = append(adjMap[key], int32(i))
		}
	}

	for i := 0; i < f.count; i++ {
		fc := f.container[i]
		if fc == nil || fc.hasExplicitNormals {
			continue
		}
		fd := faceData[i]
		if fd.nx == 0 && fd.ny == 0 && fd.nz == 0 {
			continue
		}
		for k := 0; k < 3; k++ {
			key := makeSmoothVertKey(fc.tri.Points[k])
			refs := adjMap[key]
			var sx, sy, sz float64
			for _, rIdx := range refs {
				ofd := faceData[rIdx]
				if fd.nx*ofd.nx+fd.ny*ofd.ny+fd.nz*ofd.nz >= cosCrease {
					sx += ofd.ax
					sy += ofd.ay
					sz += ofd.az
				}
			}
			fc.vertexNormals[k] = normalizeXYZ(geometry.XYZ{X: sx, Y: sy, Z: sz}, fc.n)
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
			mixF32(float32(p.X))
			mixF32(float32(p.Y))
			mixF32(float32(p.Z))
			mixF32(float32(fc.u[k]))
			mixF32(float32(fc.v[k]))
			mixF32(float32(fc.vertexNormals[k].X))
			mixF32(float32(fc.vertexNormals[k].Y))
			mixF32(float32(fc.vertexNormals[k].Z))
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
	edgeAdj := make(map[EdgeKey]edgeAdjacency, f.count*2)

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

	sinToPlane := func(opp, anchor, n geometry.XYZ) float64 {
		dx := opp.X - anchor.X
		dy := opp.Y - anchor.Y
		dz := opp.Z - anchor.Z
		lenSq := dx*dx + dy*dy + dz*dz
		if lenSq <= 1e-18 {
			return 0.0
		}
		return (dx*n.X + dy*n.Y + dz*n.Z) / math.Sqrt(lenSq)
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

		cosAngle := fa.n.X*fb.n.X + fa.n.Y*fb.n.Y + fa.n.Z*fb.n.Z
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
		// Quattro intervalli geometrici lungo il raggio d'impatto.
		targetEdge := radius * 0.25
		if targetEdge < deformMinEdge {
			targetEdge = deformMinEdge
		}
		if _, err := f.TessellateNear(impact, radius, targetEdge, deformMaxFaces); err != nil {
			return false
		}
	}
	f.TessellateInit()
	radiusSq := radius * radius

	for k := range f.displacements {
		delete(f.displacements, k)
	}

	for i := 0; i < f.count; i++ {
		fc := f.container[i]
		if fc == nil || !fc.IsDeformable() {
			continue
		}
		for _, p := range fc.tri.Points {
			key := MakeVertexKey(p)
			if _, exists := f.displacements[key]; exists {
				continue
			}
			dx := p.X - impact.X
			dy := p.Y - impact.Y
			dz := p.Z - impact.Z
			distSq := dx*dx + dy*dy + dz*dz
			if distSq >= radiusSq {
				continue
			}
			t := 1.0 - math.Sqrt(distSq)/radius
			factor := t * t * (3.0 - 2.0*t)
			f.displacements[key] = geometry.XYZ{X: impulse.X * factor, Y: impulse.Y * factor, Z: impulse.Z * factor}
		}
	}

	// Fallback per triangoli grandi i cui vertici ricadono all'esterno del raggio d'impatto
	if len(f.displacements) == 0 {
		for i := 0; i < f.count; i++ {
			fc := f.container[i]
			if fc == nil || !fc.IsDeformable() {
				continue
			}
			w0, w1, w2, inside := fc.Barycentric(impact)
			if !inside {
				continue
			}
			p0 := fc.tri.Points[0]
			nx, ny, nz := fc.GetNormal()
			distPlane := math.Abs((impact.X-p0.X)*nx + (impact.Y-p0.Y)*ny + (impact.Z-p0.Z)*nz)
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
				disp := geometry.XYZ{X: impulse.X * weight, Y: impulse.Y * weight, Z: impulse.Z * weight}
				if existing, ok := f.displacements[key]; ok {
					if weight*weight > existing.X*existing.X+existing.Y*existing.Y+existing.Z*existing.Z {
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
		if fc == nil {
			continue
		}
		for j := 0; j < geometry.Triangle3DPoints; j++ {
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

	midpoint := func(a, b geometry.XYZ) geometry.XYZ {
		return geometry.XYZ{X: (a.X + b.X) * 0.5, Y: (a.Y + b.Y) * 0.5, Z: (a.Z + b.Z) * 0.5}
	}

	distanceSquared := func(a, b geometry.XYZ) float64 {
		dx := a.X - b.X
		dy := a.Y - b.Y
		dz := a.Z - b.Z
		return dx*dx + dy*dy + dz*dz
	}

	f.TessellateInit()

	initialCount := f.count
	maxEdgeSq := maxEdge * maxEdge
	searchRadius := radius + maxEdge

	for pass := 0; pass < deformMaxSubdivisionPasses; pass++ {
		for k := range f.markedEdges {
			delete(f.markedEdges, k)
		}
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
			edges := [3][2]geometry.XYZ{{p[0], p[1]}, {p[1], p[2]}, {p[2], p[0]}}
			for _, edge := range edges {
				d2 := distanceSquared(edge[0], edge[1])
				// Non suddividere se lo spigolo è già più corto di deformMinEdge
				if d2 > maxEdgeSq && d2 > minEdgeSq {
					f.markedEdges[MakeEdgeKey(edge[0], edge[1])] = struct{}{}
				}
			}
		}
		if len(f.markedEdges) == 0 {
			break
		}
		// ogni lato condiviso riceve un unico punto medio.
		for k := range f.midpoints {
			delete(f.midpoints, k)
		}
		for i := 0; i < f.count; i++ {
			fc := f.container[i]
			if fc == nil {
				continue
			}

			p := fc.tri.Points
			edges := [3][2]geometry.XYZ{{p[0], p[1]}, {p[1], p[2]}, {p[2], p[0]}}
			for _, edge := range edges {
				key := MakeEdgeKey(edge[0], edge[1])
				if _, ok := f.markedEdges[key]; ok {
					if _, exists := f.midpoints[key]; !exists {
						f.midpoints[key] = midpoint(edge[0], edge[1])
					}
				}
			}
		}
		// applica le suddivisioni a tutte le facce incidenti.
		f.nextFaces = f.nextFaces[:0]
		for i := 0; i < f.count; i++ {
			fc := f.container[i]
			if fc == nil {
				continue
			}
			p := fc.tri.Points
			keys := [3]EdgeKey{MakeEdgeKey(p[0], p[1]), MakeEdgeKey(p[1], p[2]), MakeEdgeKey(p[2], p[0])}
			var mask uint8
			var mab, mbc, mca geometry.XYZ
			if m, ok := f.midpoints[keys[0]]; ok {
				mask |= 1
				mab = m
			}
			if m, ok := f.midpoints[keys[1]]; ok {
				mask |= 2
				mbc = m
			}
			if m, ok := f.midpoints[keys[2]]; ok {
				mask |= 4
				mca = m
			}
			children := fc.SplitFace(mask, mab, mbc, mca)
			f.nextFaces = append(f.nextFaces, children...)
			if maxFaces > 0 && len(f.nextFaces) > maxFaces {
				return f.count - initialCount, fmt.Errorf("tessellation face limit exceeded: %d", maxFaces)
			}
		}
		f.container, f.nextFaces = f.nextFaces, f.container
		f.count = len(f.container)
	}
	return f.count - initialCount, nil
}
