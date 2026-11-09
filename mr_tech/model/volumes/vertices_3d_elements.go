package volumes

import (
	"fmt"
	"math/rand"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/physics"
)

// Vertices3DElements stores face, volume, and physics state for managing 3D vertex element entries.
type Vertices3DElements struct {
	facesA          *Faces
	facesB          *Faces
	partFaceOffsets []int
	partFaceCounts  []int
	viewVolume      *Volume
	entity          *physics.Entity
	entries         []*Vertices3DEntry
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
			v.facesA.Set(idx, &Face{material: src.material, u: src.u, v: src.v})
			v.facesB.Set(idx, &Face{material: src.material, u: src.u, v: src.v})
		}
	}

	element := v.entries[0]
	v.viewVolume = element.viewVolume
	v.entity = element.VerticesGetEntity()
	return nil
}

// Rebuild reconstructs the composite view volume from source faces and updates the linked physics entity.
func (v *Vertices3DElements) Rebuild(thing IThing, cfg *config.Thing) {
	facesA := v.facesA
	compVol := NewVolume(0, fmt.Sprintf("%s_md3_composite", cfg.Id), "thing", cfg.Mass, cfg.Restitution, cfg.Friction, cfg.GForce)
	for idx := 0; idx < facesA.Len(); idx++ {
		src := facesA.At(idx)
		if src == nil {
			continue
		}
		fCopy := NewFace(src.ClonePoints(), fmt.Sprintf("%s_md3_f_%d", cfg.Id, idx), src.material)
		u0, v0 := src.GetUV0()
		u1, v1 := src.GetUV1()
		u2, v2 := src.GetUV2()
		fCopy.SetUV(u0, v0, u1, v1, u2, v2)
		fCopy.LockUV(true)
		compVol.AddFace(fCopy)
	}
	compVol.Rebuild()
	compVol.SetThing(thing)
	v.viewVolume = compVol
	v.entity = compVol.GetEntity()
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
func (v *Vertices3DElements) Compile(i int, origAX, origAY, origAZ, origBX, origBY, origBZ float64, rotA, rotB *TagAxis) {
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
			rotA.TransformPointsWithMatrix(dstA, srcA, origAX, origAY, origAZ)
		}

		srcB := volB.GetFace(j)
		dstB := v.facesB.At(offset + j)
		if isIdentB {
			if origBX == 0 && origBY == 0 && origBZ == 0 {
				dstB.CopyPoints(srcB)
			} else {
				dstB.TransformPoints(srcB, origBX, origBY, origBZ)
			}
		} else {
			rotB.TransformPointsWithMatrix(dstB, srcB, origBX, origBY, origBZ)
		}
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

// GetCenter returns the x, y, and z coordinates of the elements' current center.
func (v *Vertices3DElements) GetCenter() (float64, float64, float64) {
	return v.entity.GetCenter()
}
