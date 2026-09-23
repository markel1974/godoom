package model

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

type VerticesMD3 struct {
	lower *VerticesMD1
	upper *VerticesMD1
	head  *VerticesMD1

	facesA    []*Face
	facesB    []*Face
	facesAPtr *[]*Face
	facesBPtr *[]*Face

	totalFaces int
	entity     *physics.Entity
}

func NewVerticesMD3(cfg *config.Thing, materials *Materials) *VerticesMD3 {
	if cfg.MD3 == nil {
		panic(fmt.Sprintf("no MD3 for thing %s", cfg.Id))
	}

	cfgLower := cfg.Clone()
	cfgLower.MD1 = cfg.MD3.Lower
	cfgLower.MD3 = nil
	lower := NewVerticesMD2(cfgLower, materials)

	cfgUpper := cfg.Clone()
	cfgUpper.MD1 = cfg.MD3.Upper
	cfgUpper.MD3 = nil
	upper := NewVerticesMD2(cfgUpper, materials)

	cfgHead := cfg.Clone()
	cfgHead.MD1 = cfg.MD3.Head
	cfgHead.MD3 = nil
	head := NewVerticesMD2(cfgHead, materials)

	lowerFaces, countL := lower.volumes[0].GetFaces()
	upperFaces, countU := upper.volumes[0].GetFaces()
	headFaces, countH := head.volumes[0].GetFaces()
	totalFaces := countL + countU + countH

	v := &VerticesMD3{
		lower:      lower,
		upper:      upper,
		head:       head,
		totalFaces: totalFaces,
		facesA:     make([]*Face, totalFaces),
		facesB:     make([]*Face, totalFaces),
	}
	v.facesAPtr = &v.facesA
	v.facesBPtr = &v.facesB

	for i := 0; i < totalFaces; i++ {
		v.facesA[i] = &Face{}
		v.facesB[i] = &Face{}

		var src *Face
		if i < countL {
			src = (*lowerFaces)[i]
		} else if i < countL+countU {
			src = (*upperFaces)[i-countL]
		} else {
			src = (*headFaces)[i-countL-countU]
		}

		v.facesA[i].material = src.material
		v.facesA[i].u = src.u
		v.facesA[i].v = src.v

		v.facesB[i].material = src.material
		v.facesB[i].u = src.u
		v.facesB[i].v = src.v
	}

	v.entity = lower.GetEntity()

	return v
}

func (v *VerticesMD3) GetVolume() *Volume {
	return v.lower.viewVolume
}

func (v *VerticesMD3) GetEntity() *physics.Entity {
	return v.entity
}

func (v *VerticesMD3) GetAABB() *physics.AABB {
	return v.entity.GetAABB()
}

func (v *VerticesMD3) SetAction(idx int) {
	v.lower.SetAction(idx)
	v.upper.SetAction(idx)
	v.head.SetAction(idx)
}

func (v *VerticesMD3) GetDisplacement() (float64, float64, float64) {
	return v.lower.GetDisplacement()
}

func (v *VerticesMD3) GetBillboard() float64 {
	return v.lower.GetBillboard()
}

func (v *VerticesMD3) SetThing(t IThing) {
	v.lower.SetThing(t)
	v.upper.SetThing(t)
	v.head.SetThing(t)
}

func transformPoints(dst *Face, src *Face, origin geometry.XYZ) {
	pts := src.GetPoints()
	dst.tri[0] = geometry.XYZ{X: pts[0].X + origin.X, Y: pts[0].Y + origin.Y, Z: pts[0].Z + origin.Z}
	dst.tri[1] = geometry.XYZ{X: pts[1].X + origin.X, Y: pts[1].Y + origin.Y, Z: pts[1].Z + origin.Z}
	dst.tri[2] = geometry.XYZ{X: pts[2].X + origin.X, Y: pts[2].Y + origin.Y, Z: pts[2].Z + origin.Z}
}

func copyFacePoints(dst *Face, src *Face) {
	pts := src.GetPoints()
	dst.tri[0] = pts[0]
	dst.tri[1] = pts[1]
	dst.tri[2] = pts[2]
}

func (v *VerticesMD3) GetVertices(tick uint64) (*[]*Face, int, *[]*Face, int, float64, float64) {
	facesL_A, countL, facesL_B, _, lerpT, billboard := v.lower.GetVertices(tick)
	facesU_A, countU, facesU_B, _, _, _ := v.upper.GetVertices(tick)
	facesH_A, countH, facesH_B, _, _, _ := v.head.GetVertices(tick)

	// In the real implementation, we would extract the exact tag corresponding to the frame.
	// But GetVertices returns faces, not the Volume itself, so we can't get Tags directly
	// unless we find the Volume.
	// Since we know the frames from GetVertices logic:
	var getVol = func(md1 *VerticesMD1, t uint64) (*Volume, *Volume) {
		if md1.startFrame == md1.endFrame {
			return md1.volumes[md1.startFrame], md1.volumes[md1.startFrame]
		}
		const groupSize = 6.0
		frameFloat := textures.TickGrouped(t, int(groupSize))
		animLength := md1.endFrame - md1.startFrame + 1
		if animLength <= 0 {
			animLength = 1
		}
		relativeFrameA := int(frameFloat) % animLength
		relativeFrameB := (relativeFrameA + 1) % animLength
		return md1.volumes[md1.startFrame+relativeFrameA], md1.volumes[md1.startFrame+relativeFrameB]
	}

	volL_A, volL_B := getVol(v.lower, tick)
	volU_A, volU_B := getVol(v.upper, tick)
	_, _ = getVol(v.head, tick) // Execute to keep state synced but discard volumes

	tagTorsoA := volL_A.Tags["tag_torso"]
	tagTorsoB := volL_B.Tags["tag_torso"]

	tagHeadA := volU_A.Tags["tag_head"]
	tagHeadB := volU_B.Tags["tag_head"]

	combinedHeadA := geometry.XYZ{X: tagTorsoA.X + tagHeadA.X, Y: tagTorsoA.Y + tagHeadA.Y, Z: tagTorsoA.Z + tagHeadA.Z}
	combinedHeadB := geometry.XYZ{X: tagTorsoB.X + tagHeadB.X, Y: tagTorsoB.Y + tagHeadB.Y, Z: tagTorsoB.Z + tagHeadB.Z}

	for i := 0; i < countL; i++ {
		copyFacePoints(v.facesA[i], (*facesL_A)[i])
		copyFacePoints(v.facesB[i], (*facesL_B)[i])
	}

	for i := 0; i < countU; i++ {
		transformPoints(v.facesA[countL+i], (*facesU_A)[i], tagTorsoA)
		transformPoints(v.facesB[countL+i], (*facesU_B)[i], tagTorsoB)
	}

	for i := 0; i < countH; i++ {
		transformPoints(v.facesA[countL+countU+i], (*facesH_A)[i], combinedHeadA)
		transformPoints(v.facesB[countL+countU+i], (*facesH_B)[i], combinedHeadB)
	}

	return v.facesAPtr, v.totalFaces, v.facesBPtr, v.totalFaces, lerpT, billboard
}
