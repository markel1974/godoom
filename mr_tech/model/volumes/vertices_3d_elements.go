package volumes

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/physics"
)

type Vertices3DElements struct {
	facesA          *Faces
	facesB          *Faces
	partFaceOffsets []int
	partFaceCounts  []int
	viewVolume      *Volume
	entity          *physics.Entity
	elements        []*Vertices3DEntry
	volsA           []*Volume
	volsB           []*Volume
	lerpT           float64
}

func NewVertices3DElements() *Vertices3DElements {
	return &Vertices3DElements{}
}

func (v *Vertices3DElements) Setup(thing IThing, cfg *config.Thing, materials *materials.Materials) error {
	numParts := len(cfg.Model3D.Parts)
	if numParts == 0 {
		panic("no parts found in vertices_3d")
	}
	v.elements = make([]*Vertices3DEntry, numParts)
	v.partFaceOffsets = make([]int, numParts)
	v.partFaceCounts = make([]int, numParts)
	v.volsA = make([]*Volume, numParts)
	v.volsB = make([]*Volume, numParts)

	totalFaces := 0

	for i, part := range cfg.Model3D.Parts {
		if part == nil {
			continue
		}
		c := cfg.Clone()
		c.Model3DEntry = part
		c.Model3D = nil
		entry := NewVertices3DEntry(thing, c, materials)
		if i > 0 && v.elements[0] != nil {
			entry.salt = v.elements[0].salt
		}
		v.elements[i] = entry
		if len(entry.volumes) > 0 && entry.volumes[0] != nil {
			count := entry.volumes[0].GetFaceCount()
			v.partFaceOffsets[i] = totalFaces
			v.partFaceCounts[i] = count
			totalFaces += count
		}
	}

	v.facesA = NewFaces(totalFaces, true)
	v.facesB = NewFaces(totalFaces, true)

	// Initialize face buffers using partFaceOffsets to decouple buffer layout from execution order
	for i, el := range v.elements {
		if el == nil {
			panic("nil element found in vertices_3d")
		}
		if len(el.volumes) == 0 {
			panic("nil volumes found in vertices_3d")
		}
		if el.volumes[0] == nil {
			panic("nil volumes[0] found in vertices_3d")
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

	element := v.elements[0]
	v.viewVolume = element.viewVolume
	v.entity = element.VerticesGetEntity()
	return nil
}

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

func (v *Vertices3DElements) Bind(i int, tick uint64) {
	element := v.elements[i]
	_ = element.VerticesCompile(tick)
	volA, volB, lerpT := element.GetVolumes()
	if i == 0 {
		v.lerpT = lerpT
	}
	v.volsA[i] = volA
	v.volsB[i] = volB
}

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

func (v *Vertices3DElements) Len() int {
	return len(v.elements)
}

func (v *Vertices3DElements) GetElement(i int) *Vertices3DEntry {
	if i >= 0 && i < len(v.elements) {
		return v.elements[i]
	}
	return nil
}

func (v *Vertices3DElements) Freeze() {
	for _, el := range v.elements {
		if el != nil {
			el.FreezeCurrentFrame()
		}
	}
}

func (v *Vertices3DElements) GetCenter() (float64, float64, float64) {
	return v.entity.GetCenter()
}
