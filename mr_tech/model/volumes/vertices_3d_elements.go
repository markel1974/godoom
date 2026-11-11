package volumes

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
)

type VirtualVolume struct {
	entity   *physics.Entity
	elements *Vertices3DElements
}

func NewVirtualVolume(elements *Vertices3DElements, entity *physics.Entity) *VirtualVolume {
	return &VirtualVolume{
		elements: elements,
		entity:   entity,
	}
}

func (v *VirtualVolume) GetEntity() *physics.Entity {
	return v.entity
}

// QueryOverlaps interroga i volumi attivi di tutte le parti che compongono Vertices3D
func (v *VirtualVolume) QueryOverlaps(object aabb.IObject, callback func(object aabb.IObject) bool) {
	for _, el := range v.elements.entries {
		el.comp2.QueryOverlaps(object, callback)
	}
}

// Vertices3DElements stores face, volume, and physics state for managing 3D vertex element entries.
type Vertices3DElements struct {
	facesA          *Faces
	facesB          *Faces
	partFaceOffsets []int
	partFaceCounts  []int
	entries         []*Vertices3DEntry
	vVolume         *VirtualVolume
	volsA           []*Volume
	volsB           []*Volume
	lerpT           float64
}

// NewVertices3DElements creates an empty Vertices3DElements collection.
func NewVertices3DElements() *Vertices3DElements {
	return &Vertices3DElements{}
}

// Setup initializes 3D vertex elements, face buffers, view volume, and entity from the given configuration.
func (v *Vertices3DElements) Setup(thing IThing, cfg *config.Thing, materials *materials.Materials) error {
	numParts := len(cfg.Model3D.Parts)
	if numParts == 0 {
		return fmt.Errorf("no parts found in vertices_3d")
	}
	v.entries = make([]*Vertices3DEntry, numParts)
	v.partFaceOffsets = make([]int, numParts)
	v.partFaceCounts = make([]int, numParts)
	v.volsA = make([]*Volume, numParts)
	v.volsB = make([]*Volume, numParts)

	totalFaces := 0

	salt := uint64(rand.Intn(512))

	for i, part := range cfg.Model3D.Parts {
		if part == nil {
			return fmt.Errorf("nil part found in vertices_3d")
		}
		c := cfg.Clone()
		c.Model3DEntry = part
		c.Model3D = nil
		entry := NewVertices3DEntry(thing, c, materials)
		entry.salt = salt
		v.entries[i] = entry
		if len(entry.volumes) == 0 {
			return fmt.Errorf("empty volumes found in vertices_3d")
		}
		vol := entry.volumes[0]
		if vol == nil {
			return fmt.Errorf("nil volumes[0] found in vertices_3d")
		}
		count := vol.GetFaceCount()
		v.partFaceOffsets[i] = totalFaces
		v.partFaceCounts[i] = count
		totalFaces += count
	}

	v.facesA = NewFaces(totalFaces, true)
	v.facesB = NewFaces(totalFaces, true)

	// Initialize face buffers using partFaceOffsets to decouple buffer layout from execution order
	for i, el := range v.entries {
		if el == nil {
			return fmt.Errorf("nil element found in vertices_3d")
		}
		if len(el.volumes) == 0 {
			return fmt.Errorf("nil volumes found in vertices_3d")
		}
		if el.volumes[0] == nil {
			return fmt.Errorf("nil volumes[0] found in vertices_3d")
		}
		target := el.volumes[0]
		count := target.GetFaceCount()
		offset := v.partFaceOffsets[i]
		for j := 0; j < count; j++ {
			src := target.GetFace(j)
			idx := offset + j
			v.facesA.Set(idx, NewFaceEmpty(src.material, src.u, src.v))
			v.facesB.Set(idx, NewFaceEmpty(src.material, src.u, src.v))
		}
	}

	v.vVolume = NewVirtualVolume(v, physics.NewEntity(cfg.Mass, cfg.Restitution, cfg.Friction, cfg.GForce))
	return nil
}

// Bind compiles element i for tick, caches its volumes, and records the first element's interpolation factor.
func (v *Vertices3DElements) Bind(i int, tick uint64) {
	element := v.entries[i]
	_ = element.VerticesCompile(tick)
	volA, volB, lerpT := element.GetVolumes()
	if i == 0 {
		v.lerpT = lerpT
	}
	v.volsA[i] = volA
	v.volsB[i] = volB
}

// Compile transforms frame faces for part i into the A and B buffers and clears unused face slots.
func (v *Vertices3DElements) Compile(i int, origAX, origAY, origAZ, origBX, origBY, origBZ float64, rotA, rotB *physics.TagAxis) (float64, float64, float64, float64, float64, float64) {
	volA := v.volsA[i]
	volB := v.volsB[i]

	isIdentA := rotA.IsIdentity()
	isIdentB := rotB.IsIdentity()

	offset := v.partFaceOffsets[i]
	partCount := v.partFaceCounts[i]

	countA := volA.GetFaceCount()
	countB := volB.GetFaceCount()

	count := countA
	if countB < count {
		count = countB
	}
	if partCount < count {
		count = partCount
	}

	minX, minY, minZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
	maxX, maxY, maxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64

	for j := 0; j < count; j++ {
		srcA := volA.GetFace(j)
		dstA := v.facesA.At(offset + j)
		if isIdentA {
			if origAX == 0 && origAY == 0 && origAZ == 0 {
				dstA.CopyPoints(srcA)
			} else {
				dstA.TransformPoints(srcA, origAX, origAY, origAZ)
			}
		} else {
			rotA.TransformPointsWithMatrix(dstA.tri, srcA.tri, origAX, origAY, origAZ)
		}
		minX, minY, minZ, maxX, maxY, maxZ = dstA.GetMinMax(minX, minY, minZ, maxX, maxY, maxZ)

		srcB := volB.GetFace(j)
		dstB := v.facesB.At(offset + j)
		if isIdentB {
			if origBX == 0 && origBY == 0 && origBZ == 0 {
				dstB.CopyPoints(srcB)
			} else {
				dstB.TransformPoints(srcB, origBX, origBY, origBZ)
			}
		} else {
			rotB.TransformPointsWithMatrix(dstB.tri, srcB.tri, origBX, origBY, origBZ)
		}
		minX, minY, minZ, maxX, maxY, maxZ = dstB.GetMinMax(minX, minY, minZ, maxX, maxY, maxZ)
	}

	// Zero out any leftover faces in this part's buffer slot to avoid displaying stale frame data
	for j := count; j < partCount; j++ {
		if faceA := v.facesA.At(offset + j); faceA != nil {
			faceA.Zero()
		}
		if faceB := v.facesB.At(offset + j); faceB != nil {
			faceB.Zero()
		}
	}

	return minX, minY, minZ, maxX, maxY, maxZ
}

// Len returns the number of 3D vertex elements.
func (v *Vertices3DElements) Len() int {
	return len(v.entries)
}

// GetEntry returns the vertices entry at index i, or nil if i is out of range.
func (v *Vertices3DElements) GetEntry(i int) *Vertices3DEntry {
	if i >= 0 && i < len(v.entries) {
		return v.entries[i]
	}
	return nil
}

// Freeze locks all non-nil 3D elements on their current animation frame.
func (v *Vertices3DElements) Freeze() {
	for _, el := range v.entries {
		if el != nil {
			el.FreezeCurrentFrame()
		}
	}
}

// UpdateEntityBounds sets the entity size from current face bounds when all dimensions are positive.
func (v *Vertices3DElements) UpdateEntityBounds() {
	minX, minY, minZ, maxX, maxY, maxZ := v.facesA.GetMinMax()
	w := maxX - minX
	h := maxY - minY
	d := maxZ - minZ
	v.vVolume.entity.SetSize(w, h, d)
}
