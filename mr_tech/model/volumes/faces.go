package volumes

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

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

	markedEdges   map[edgeKey]struct{}
	midpoints     map[edgeKey]geometry.XYZ
	displacements map[vertexKey]geometry.XYZ
	nextFaces     []*Face
}

// NewFaces creates a Faces collection with the given initial capacity.
// If fill is true, the collection length is initialized to initSize.
func NewFaces(initSize int, fill bool) *Faces {
	count := 0
	if fill {
		count = initSize
	}
	return &Faces{
		tree:          aabb.NewTree(64, 0.0),
		container:     make([]*Face, initSize),
		count:         count,
		markedEdges:   make(map[edgeKey]struct{}, 128),
		midpoints:     make(map[edgeKey]geometry.XYZ, 128),
		displacements: make(map[vertexKey]geometry.XYZ, 128),
		nextFaces:     make([]*Face, 0, initSize),
	}
}

func (f *Faces) initBuffers() {
	if f.markedEdges == nil {
		f.markedEdges = make(map[edgeKey]struct{}, 128)
	}
	if f.midpoints == nil {
		f.midpoints = make(map[edgeKey]geometry.XYZ, 128)
	}
	if f.displacements == nil {
		f.displacements = make(map[vertexKey]geometry.XYZ, 128)
	}
}

// Rebuild recomputes cached centroid and inertia values from the current faces.
func (f *Faces) Rebuild() {
	f.tree.Clear()
	if f.count == 0 {
		f.minX, f.minY, f.minZ, f.maxX, f.maxY, f.maxZ = 0, 0, 0, 0, 0, 0
		f.w, f.h, f.d = 0, 0, 0
		f.cx, f.cy, f.cz = 0, 0, 0
		f.ixy, f.ixz, f.iyy, f.iyz, f.izz = 0, 0, 0, 0, 0
		return
	}
	f.minX, f.minY, f.minZ = math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
	f.maxX, f.maxY, f.maxZ = -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64
	for x := 0; x < f.count; x++ {
		face := f.container[x]
		face.Rebuild()
		f.tree.InsertObject(face)
		f.minX, f.minY, f.minZ, f.maxX, f.maxY, f.maxZ = face.GetMinMax(f.minX, f.minY, f.minZ, f.maxX, f.maxY, f.maxZ)
	}
	f.cx, f.cy, f.cz, f.ixx, f.ixy, f.ixz, f.iyy, f.iyz, f.izz = f.ComputeMassProperties()
	f.w = f.maxX - f.minX
	f.h = f.maxY - f.minY
	f.d = f.maxZ - f.minZ
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
		newFaces := make([]*Face, f.count*2)
		copy(newFaces, f.container)
		f.container = newFaces
	}
	f.container[f.count] = face
	f.count++
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
}

// At returns the face stored at idx.
func (f *Faces) At(idx int) *Face {
	return f.container[idx]
}

// Clear removes all faces from the collection.
func (f *Faces) Clear() {
	f.tree.Clear()
	f.count = 0
}

func (f *Faces) Deform(impact geometry.XYZ, impulse geometry.XYZ, radius float64) bool {
	// Quattro intervalli geometrici lungo il raggio d'impatto.
	targetEdge := radius * 0.25
	if targetEdge < deformMinEdge {
		targetEdge = deformMinEdge
	}
	if _, err := f.TessellateNear(impact, radius, targetEdge, deformMaxFaces); err != nil {
		return false
	}
	radiusSq := radius * radius

	f.initBuffers()
	for k := range f.displacements {
		delete(f.displacements, k)
	}

	for i := 0; i < f.count; i++ {
		fc := f.container[i]
		if fc == nil || !fc.IsDeformable() {
			continue
		}
		for _, p := range fc.tri.Points {
			key := makeVertexKey(p)
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
			if displacement, ok := f.displacements[makeVertexKey(*p)]; ok {
				p.X += displacement.X
				p.Y += displacement.Y
				p.Z += displacement.Z
			}
		}
	}

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

	f.initBuffers()

	initialCount := f.count
	maxEdgeSq := maxEdge * maxEdge
	minEdgeSq := deformMinEdge * deformMinEdge
	searchRadius := radius + maxEdge

	for pass := 0; pass < deformMaxSubdivisionPasses; pass++ {
		for k := range f.markedEdges {
			delete(f.markedEdges, k)
		}
		// Fase 1: individua gli spigoli da raffinare nella regione d'impatto.
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
					f.markedEdges[makeEdgeKey(edge[0], edge[1])] = struct{}{}
				}
			}
		}
		if len(f.markedEdges) == 0 {
			break
		}
		// Fase 2: ogni lato condiviso riceve un unico punto medio.
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
				key := makeEdgeKey(edge[0], edge[1])
				if _, ok := f.markedEdges[key]; ok {
					if _, exists := f.midpoints[key]; !exists {
						f.midpoints[key] = midpoint(edge[0], edge[1])
					}
				}
			}
		}
		// Fase 3: applica le suddivisioni a tutte le facce incidenti.
		f.nextFaces = f.nextFaces[:0]
		for i := 0; i < f.count; i++ {
			fc := f.container[i]
			if fc == nil {
				continue
			}
			p := fc.tri.Points
			keys := [3]edgeKey{makeEdgeKey(p[0], p[1]), makeEdgeKey(p[1], p[2]), makeEdgeKey(p[2], p[0])}
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
			children := fc.splitFace(mask, mab, mbc, mca)
			f.nextFaces = append(f.nextFaces, children...)
			if maxFaces > 0 && len(f.nextFaces) > maxFaces {
				return f.count - initialCount,
					fmt.Errorf("tessellation face limit exceeded: %d", maxFaces)
			}
		}
		f.container, f.nextFaces = f.nextFaces, f.container
		f.count = len(f.container)
	}
	return f.count - initialCount, nil
}

// splitFace subdivides a triangular face along masked edges using supplied midpoints and interpolated UVs.
// Edge mask bits select AB, BC, and CA; a zero mask returns the original face.
func (fc *Face) splitFace(mask uint8, mab, mbc, mca geometry.XYZ) []*Face {
	if mask == 0 {
		return []*Face{fc}
	}
	averageUV := func(a, b float64) float64 {
		return (a + b) * 0.5
	}
	a := fc.tri.Points[0]
	b := fc.tri.Points[1]
	c := fc.tri.Points[2]

	u, v := fc.u, fc.v

	uvAB := [2]float64{averageUV(u[0], u[1]), averageUV(v[0], v[1])}
	uvBC := [2]float64{averageUV(u[1], u[2]), averageUV(v[1], v[2])}
	uvCA := [2]float64{averageUV(u[2], u[0]), averageUV(v[2], v[0])}
	newFace := fc.NewFaceSubdivided

	switch mask {
	case 1:
		return []*Face{
			newFace([3]geometry.XYZ{a, mab, c}, [3]float64{u[0], uvAB[0], u[2]}, [3]float64{v[0], uvAB[1], v[2]}),
			newFace([3]geometry.XYZ{mab, b, c}, [3]float64{uvAB[0], u[1], u[2]}, [3]float64{uvAB[1], v[1], v[2]}),
		}
	case 2:
		return []*Face{
			newFace([3]geometry.XYZ{a, b, mbc}, [3]float64{u[0], u[1], uvBC[0]}, [3]float64{v[0], v[1], uvBC[1]}),
			newFace([3]geometry.XYZ{a, mbc, c}, [3]float64{u[0], uvBC[0], u[2]}, [3]float64{v[0], uvBC[1], v[2]}),
		}
	case 4:
		return []*Face{
			newFace([3]geometry.XYZ{a, b, mca}, [3]float64{u[0], u[1], uvCA[0]}, [3]float64{v[0], v[1], uvCA[1]}),
			newFace([3]geometry.XYZ{mca, b, c}, [3]float64{uvCA[0], u[1], u[2]}, [3]float64{uvCA[1], v[1], v[2]}),
		}
	case 3:
		return []*Face{
			newFace([3]geometry.XYZ{b, mbc, mab}, [3]float64{u[1], uvBC[0], uvAB[0]}, [3]float64{v[1], uvBC[1], uvAB[1]}),
			newFace([3]geometry.XYZ{a, mab, c}, [3]float64{u[0], uvAB[0], u[2]}, [3]float64{v[0], uvAB[1], v[2]}),
			newFace([3]geometry.XYZ{mab, mbc, c}, [3]float64{uvAB[0], uvBC[0], u[2]}, [3]float64{uvAB[1], uvBC[1], v[2]}),
		}
	case 5:
		return []*Face{
			newFace([3]geometry.XYZ{a, mab, mca}, [3]float64{u[0], uvAB[0], uvCA[0]}, [3]float64{v[0], uvAB[1], uvCA[1]}),
			newFace([3]geometry.XYZ{mab, b, c}, [3]float64{uvAB[0], u[1], u[2]}, [3]float64{uvAB[1], v[1], v[2]}),
			newFace([3]geometry.XYZ{mab, c, mca}, [3]float64{uvAB[0], u[2], uvCA[0]}, [3]float64{uvAB[1], v[2], uvCA[1]}),
		}
	case 6:
		return []*Face{
			newFace([3]geometry.XYZ{c, mca, mbc}, [3]float64{u[2], uvCA[0], uvBC[0]}, [3]float64{v[2], uvCA[1], uvBC[1]}),
			newFace([3]geometry.XYZ{a, b, mbc}, [3]float64{u[0], u[1], uvBC[0]}, [3]float64{v[0], v[1], uvBC[1]}),
			newFace([3]geometry.XYZ{a, mbc, mca}, [3]float64{u[0], uvBC[0], uvCA[0]}, [3]float64{v[0], uvBC[1], uvCA[1]}),
		}
	default: // mask == 7
		return []*Face{
			newFace([3]geometry.XYZ{a, mab, mca}, [3]float64{u[0], uvAB[0], uvCA[0]}, [3]float64{v[0], uvAB[1], uvCA[1]}),
			newFace([3]geometry.XYZ{mab, b, mbc}, [3]float64{uvAB[0], u[1], uvBC[0]}, [3]float64{uvAB[1], v[1], uvBC[1]}),
			newFace([3]geometry.XYZ{mca, mbc, c}, [3]float64{uvCA[0], uvBC[0], u[2]}, [3]float64{uvCA[1], uvBC[1], v[2]}),
			newFace([3]geometry.XYZ{mab, mbc, mca}, [3]float64{uvAB[0], uvBC[0], uvCA[0]}, [3]float64{uvAB[1], uvBC[1], uvCA[1]}),
		}
	}
}
