package model

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
)

// Vertices3D represents a 3D structure composed of faces, volumes, origins, and associated metadata for spatial modeling.
type Vertices3D struct {
	facesA         []*Face
	facesB         []*Face
	facesAPtr      *[]*Face
	facesBPtr      *[]*Face
	totalFaces     int
	entity         *physics.Entity
	currentAction  int
	actionMaps     [][]int
	elements       []*Vertices3DEntry
	links          []config.Model3DLink
	linkTagIndices []int
	executionOrder []int
	volsA          []*Volume
	volsB          []*Volume
	originsA       []geometry.XYZ
	originsB       []geometry.XYZ
}

// NewVertices3D initializes and returns a new Vertices3D object based on the provided Thing configuration and materials.
func NewVertices3D(cfg *config.Thing, materials *Materials) *Vertices3D {
	if cfg.Model3D == nil {
		panic(fmt.Sprintf("no Model3D for thing %s", cfg.Id))
	}

	elements := make([]*Vertices3DEntry, len(cfg.Model3D.Parts))
	var totalFaces int

	for i, part := range cfg.Model3D.Parts {
		if part != nil {
			c := cfg.Clone()
			c.Model3DEntry = part
			c.Model3D = nil
			entry := NewVertices3DEntry(c, materials)
			elements[i] = entry
			_, count := entry.volumes[0].GetFaces()
			totalFaces += count
		}
	}

	actionMaps := cfg.Model3D.ActionMaps
	links := cfg.Model3D.Links

	linkTagIndices := make([]int, len(links))
	for i, link := range links {
		if link.Parent >= 0 && link.Parent < len(elements) && elements[link.Parent] != nil {
			linkTagIndices[i] = elements[link.Parent].GetTagIndex(link.Tag)
		} else {
			linkTagIndices[i] = -1
		}
	}

	// Build topological execution order based on Links using a Depth-First Search (DFS).
	// This ensures that parents are always processed before their children,
	// allowing us to calculate global origins in a single flat loop regardless
	// of the order in which the parts were defined in the configuration.
	executionOrder := make([]int, 0, len(elements))
	visited := make([]bool, len(elements))
	var visit func(int)
	visit = func(node int) {
		if node < 0 || node >= len(elements) || visited[node] {
			return
		}
		// Visit the parent first (DFS traversal)
		pIdx := links[node].Parent
		if pIdx >= 0 && pIdx < len(elements) {
			visit(pIdx)
		}
		visited[node] = true
		executionOrder = append(executionOrder, node)
	}

	// Ensure all nodes are visited (including disconnected sub-trees)
	for i := range elements {
		if !visited[i] {
			visit(i)
		}
	}

	v := &Vertices3D{
		elements:       elements,
		totalFaces:     totalFaces,
		currentAction:  -1,
		actionMaps:     actionMaps,
		facesA:         make([]*Face, totalFaces),
		facesB:         make([]*Face, totalFaces),
		links:          links,
		linkTagIndices: linkTagIndices,
		executionOrder: executionOrder,
	}
	v.facesAPtr = &v.facesA
	v.facesBPtr = &v.facesB

	v.volsA = make([]*Volume, len(v.elements))
	v.volsB = make([]*Volume, len(v.elements))
	v.originsA = make([]geometry.XYZ, len(v.elements))
	v.originsB = make([]geometry.XYZ, len(v.elements))

	offset := 0
	for _, el := range elements {
		if el == nil {
			continue
		}
		faces, count := el.volumes[0].GetFaces()
		for j := 0; j < count; j++ {
			src := (*faces)[j]

			v.facesA[offset+j] = &Face{}
			v.facesB[offset+j] = &Face{}

			v.facesA[offset+j].material = src.material
			v.facesA[offset+j].u = src.u
			v.facesA[offset+j].v = src.v

			v.facesB[offset+j].material = src.material
			v.facesB[offset+j].u = src.u
			v.facesB[offset+j].v = src.v
		}
		offset += count
	}

	v.entity = elements[0].GetEntity()

	return v
}

// GetVolume retrieves the first Volume instance associated with the Vertices3D object.
func (v *Vertices3D) GetVolume() *Volume {
	return v.elements[0].viewVolume
}

// GetEntity retrieves the associated physics.Entity instance of the Vertices3D object.
func (v *Vertices3D) GetEntity() *physics.Entity {
	return v.entity
}

// GetAABB returns the axis-aligned bounding box (AABB) of the `Vertices3D` instance by delegating to its associated entity.
func (v *Vertices3D) GetAABB() *aabb.AABB {
	return v.entity.GetAABB()
}

// SetAction updates the current action of Vertices3D and propagates the action change to all associated elements.
func (v *Vertices3D) SetAction(idx int) {
	if v.currentAction == idx {
		return
	}
	v.currentAction = idx
	for _, i := range v.executionOrder {
		el := v.elements[i]
		if el != nil {
			if i < len(v.actionMaps) && idx >= 0 && idx < len(v.actionMaps[i]) {
				el.SetAction(v.actionMaps[i][idx])
			} else {
				el.SetAction(0)
			}
		}
	}
}

// GetDisplacement returns the displacement vector (dx, dy, dz) of the first element in the Vertices3D instance.
func (v *Vertices3D) GetDisplacement() (float64, float64, float64) {
	return v.elements[0].GetDisplacement()
}

// GetRenderMode returns the current render mode as a float64, used to determine how the 3D model is rendered.
func (v *Vertices3D) GetRenderMode() float64 {
	return RenderModeModel3D
}

// SetThing assigns the provided IThing instance to all non-nil entries in the elements slice of Vertices3D.
func (v *Vertices3D) SetThing(t IThing) {
	for _, e := range v.elements {
		if e != nil {
			e.SetThing(t)
		}
	}
}

// GetVertices computes and returns transformed vertex data for a specified tick, including faces, counts, interpolation, and render mode.
func (v *Vertices3D) GetVertices(tick uint64) (*[]*Face, int, *[]*Face, int, float64, float64) {
	var lerpTRet float64
	offset := 0

	for _, i := range v.executionOrder {
		el := v.elements[i]
		if el == nil {
			v.volsA[i] = nil
			v.volsB[i] = nil
			v.originsA[i].X = 0
			v.originsA[i].Y = 0
			v.originsA[i].Z = 0
			v.originsB[i].X = 0
			v.originsB[i].Y = 0
			v.originsB[i].Z = 0
			continue
		}

		volA, volB, lerpT := el.GetVolumesAt(tick)
		v.volsA[i] = volA
		v.volsB[i] = volB

		if i == v.executionOrder[0] {
			lerpTRet = lerpT
		}

		pIdx := v.links[i].Parent
		if pIdx >= 0 && pIdx < len(v.elements) && v.volsA[pIdx] != nil && v.volsB[pIdx] != nil {
			tagIdx := v.linkTagIndices[i]
			tagA, _ := v.volsA[pIdx].GetVertexTag(tagIdx)
			tagB, _ := v.volsB[pIdx].GetVertexTag(tagIdx)

			v.originsA[i].X = v.originsA[pIdx].X + tagA.X
			v.originsA[i].Y = v.originsA[pIdx].Y + tagA.Y
			v.originsA[i].Z = v.originsA[pIdx].Z + tagA.Z

			v.originsB[i].X = v.originsB[pIdx].X + tagB.X
			v.originsB[i].Y = v.originsB[pIdx].Y + tagB.Y
			v.originsB[i].Z = v.originsB[pIdx].Z + tagB.Z
		} else {
			v.originsA[i].X = 0
			v.originsA[i].Y = 0
			v.originsA[i].Z = 0
			v.originsB[i].X = 0
			v.originsB[i].Y = 0
			v.originsB[i].Z = 0
		}

		facesA, count := volA.GetFaces()
		facesB, _ := volB.GetFaces()

		if v.originsA[i].X == 0 && v.originsA[i].Y == 0 && v.originsA[i].Z == 0 && v.originsB[i].X == 0 && v.originsB[i].Y == 0 && v.originsB[i].Z == 0 {
			for j := 0; j < count; j++ {
				v3dCopyFacePoints(v.facesA[offset+j], (*facesA)[j])
				v3dCopyFacePoints(v.facesB[offset+j], (*facesB)[j])
			}
		} else {
			for j := 0; j < count; j++ {
				v3dTransformPoints(v.facesA[offset+j], (*facesA)[j], v.originsA[i])
				v3dTransformPoints(v.facesB[offset+j], (*facesB)[j], v.originsB[i])
			}
		}
		offset += count
	}

	return v.facesAPtr, v.totalFaces, v.facesBPtr, v.totalFaces, lerpTRet, v.GetRenderMode()
}

// v3dTransformPoints transforms the points of `src` Face to a new position relative to `origin`, storing them in `dst`.
func v3dTransformPoints(dst *Face, src *Face, origin geometry.XYZ) {
	dst.tri[0].X = src.tri[0].X + origin.X
	dst.tri[0].Y = src.tri[0].Y + origin.Y
	dst.tri[0].Z = src.tri[0].Z + origin.Z
	dst.tri[1].X = src.tri[1].X + origin.X
	dst.tri[1].Y = src.tri[1].Y + origin.Y
	dst.tri[1].Z = src.tri[1].Z + origin.Z
	dst.tri[2].X = src.tri[2].X + origin.X
	dst.tri[2].Y = src.tri[2].Y + origin.Y
	dst.tri[2].Z = src.tri[2].Z + origin.Z
}

// v3dCopyFacePoints copies the vertex coordinates from the source Face to the destination Face.
func v3dCopyFacePoints(dst *Face, src *Face) {
	dst.tri[0].X = src.tri[0].X
	dst.tri[0].Y = src.tri[0].Y
	dst.tri[0].Z = src.tri[0].Z
	dst.tri[1].X = src.tri[1].X
	dst.tri[1].Y = src.tri[1].Y
	dst.tri[1].Z = src.tri[1].Z
	dst.tri[2].X = src.tri[2].X
	dst.tri[2].Y = src.tri[2].Y
	dst.tri[2].Z = src.tri[2].Z
}
