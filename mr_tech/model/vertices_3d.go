package model

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
)

// Vertices3D represents a structure containing 3D vertices, along with metadata, actions, faces, and associated entities.
type Vertices3D struct {
	facesA    []*Face
	facesB    []*Face
	facesAPtr *[]*Face
	facesBPtr *[]*Face

	totalFaces    int
	entity        *physics.Entity
	currentAction int
	actionMaps    [][]int
	elements      []*Vertices3DEntry
	links         []config.Model3DLink
	volsA         []*Volume
	volsB         []*Volume
	originsA      []geometry.XYZ
	originsB      []geometry.XYZ
}

// NewVertices3D constructs and initializes a Vertices3D object using the provided configuration and materials.
// It processes the model parts (lower, upper, head, weapon) and combines their face data into a unified structure.
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

	v := &Vertices3D{
		elements:      elements,
		totalFaces:    totalFaces,
		currentAction: -1,
		actionMaps:    actionMaps,
		facesA:        make([]*Face, totalFaces),
		facesB:        make([]*Face, totalFaces),
		links:         links,
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

// GetVolume returns the Volume associated with the lower component of the Vertices3D object.
func (v *Vertices3D) GetVolume() *Volume {
	return v.elements[0].viewVolume
}

// GetEntity retrieves the physics.Entity instance associated with the Vertices3D.
func (v *Vertices3D) GetEntity() *physics.Entity {
	return v.entity
}

// GetAABB retrieves the axis-aligned bounding box (AABB) of the entity associated with the Vertices3D instance.
func (v *Vertices3D) GetAABB() *physics.AABB {
	return v.entity.GetAABB()
}

// SetAction updates the current action index of the 3D model and propagates the action change to all associated components.
func (v *Vertices3D) SetAction(idx int) {
	if v.currentAction == idx {
		return
	}
	v.currentAction = idx
	for i, el := range v.elements {
		if el != nil {
			if i < len(v.actionMaps) && idx >= 0 && idx < len(v.actionMaps[i]) {
				el.SetAction(v.actionMaps[i][idx])
			} else {
				el.SetAction(0)
			}
		}
	}
}

// GetDisplacement retrieves the 3D displacement components (dx, dy, dz) from the lower Vertices3DEntry instance.
func (v *Vertices3D) GetDisplacement() (float64, float64, float64) {
	return v.elements[0].GetDisplacement()
}

// GetRenderMode retrieves the render mode value associated with the Vertices3D instance by proxying to its lower entry.
func (v *Vertices3D) GetRenderMode() float64 {
	return v.elements[0].GetRenderMode()
}

// SetThing assigns the specified IThing instance to all associated Vertices3DEntry components.
func (v *Vertices3D) SetThing(t IThing) {
	for _, e := range v.elements {
		if e != nil {
			e.SetThing(t)
		}
	}
}

// GetVertices retrieves vertex data for rendering at a given tick, iterating dynamically through all elements.
func (v *Vertices3D) GetVertices(tick uint64) (*[]*Face, int, *[]*Face, int, float64, float64) {
	for i, el := range v.elements {
		if el != nil {
			v.volsA[i], v.volsB[i] = el.GetVolumesAt(tick)
		} else {
			v.volsA[i], v.volsB[i] = nil, nil
		}
	}

	for i := 0; i < len(v.elements); i++ {
		pIdx := v.links[i].Parent
		if pIdx >= 0 && pIdx < len(v.elements) && v.volsA[pIdx] != nil && v.volsB[pIdx] != nil {
			tagA, _ := v.volsA[pIdx].GetVertexTag(v.links[i].Tag)
			tagB, _ := v.volsB[pIdx].GetVertexTag(v.links[i].Tag)

			v.originsA[i] = geometry.XYZ{X: v.originsA[pIdx].X + tagA.X, Y: v.originsA[pIdx].Y + tagA.Y, Z: v.originsA[pIdx].Z + tagA.Z}
			v.originsB[i] = geometry.XYZ{X: v.originsB[pIdx].X + tagB.X, Y: v.originsB[pIdx].Y + tagB.Y, Z: v.originsB[pIdx].Z + tagB.Z}
		} else {
			v.originsA[i] = geometry.XYZ{}
			v.originsB[i] = geometry.XYZ{}
		}
	}

	var lerpTRet float64
	var renderModeRet float64
	offset := 0

	for i, el := range v.elements {
		if el == nil {
			continue
		}
		facesA, count, facesB, _, lerpT, renderMode := el.GetVertices(tick)
		if i == 0 {
			lerpTRet = lerpT
			renderModeRet = renderMode
		}

		for j := 0; j < count; j++ {
			if v.originsA[i].X == 0 && v.originsA[i].Y == 0 && v.originsA[i].Z == 0 && v.originsB[i].X == 0 && v.originsB[i].Y == 0 && v.originsB[i].Z == 0 {
				v3dCopyFacePoints(v.facesA[offset+j], (*facesA)[j])
				v3dCopyFacePoints(v.facesB[offset+j], (*facesB)[j])
			} else {
				v3dTransformPoints(v.facesA[offset+j], (*facesA)[j], v.originsA[i])
				v3dTransformPoints(v.facesB[offset+j], (*facesB)[j], v.originsB[i])
			}
		}
		offset += count
	}

	return v.facesAPtr, v.totalFaces, v.facesBPtr, v.totalFaces, lerpTRet, renderModeRet
}

// v3dTransformPoints transforms the points of the `src` Face by adding the `origin` offset and stores the results in `dst`.
func v3dTransformPoints(dst *Face, src *Face, origin geometry.XYZ) {
	pts := src.GetPoints()
	dst.tri[0] = geometry.XYZ{X: pts[0].X + origin.X, Y: pts[0].Y + origin.Y, Z: pts[0].Z + origin.Z}
	dst.tri[1] = geometry.XYZ{X: pts[1].X + origin.X, Y: pts[1].Y + origin.Y, Z: pts[1].Z + origin.Z}
	dst.tri[2] = geometry.XYZ{X: pts[2].X + origin.X, Y: pts[2].Y + origin.Y, Z: pts[2].Z + origin.Z}
}

// v3dCopyFacePoints copies the vertices of the triangle from the source Face to the destination Face.
func v3dCopyFacePoints(dst *Face, src *Face) {
	pts := src.GetPoints()
	dst.tri[0] = pts[0]
	dst.tri[1] = pts[1]
	dst.tri[2] = pts[2]
}
