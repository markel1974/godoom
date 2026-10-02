package model

import (
	"fmt"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
)

const (
	LowerIdx = iota
	UpperIdx
	HeadIdx
	WeaponIdx
)

// Vertices3D represents a structure containing 3D vertices, along with metadata, actions, faces, and associated entities.
type Vertices3D struct {
	facesA    []*Face
	facesB    []*Face
	facesAPtr *[]*Face
	facesBPtr *[]*Face

	totalFaces     int
	entity         *physics.Entity
	currentAction  int
	upperActionMap []int
	elements       []*Vertices3DEntry
}

// NewVertices3D constructs and initializes a Vertices3D object using the provided configuration and materials.
// It processes the model parts (lower, upper, head, weapon) and combines their face data into a unified structure.
func NewVertices3D(cfg *config.Thing, materials *Materials) *Vertices3D {
	if cfg.Model3D == nil {
		panic(fmt.Sprintf("no Model3D for thing %s", cfg.Id))
	}

	cfgLower := cfg.Clone()
	cfgLower.Model3DEntry = cfg.Model3D.Lower
	cfgLower.Model3D = nil
	lower := NewVertices3DEntry(cfgLower, materials)

	cfgUpper := cfg.Clone()
	cfgUpper.Model3DEntry = cfg.Model3D.Upper
	cfgUpper.Model3D = nil
	upper := NewVertices3DEntry(cfgUpper, materials)

	cfgHead := cfg.Clone()
	cfgHead.Model3DEntry = cfg.Model3D.Head
	cfgHead.Model3D = nil
	head := NewVertices3DEntry(cfgHead, materials)

	var weapon *Vertices3DEntry
	var countW int
	var weaponFaces *[]*Face
	if cfg.Model3D.Weapon != nil {
		cfgWeapon := cfg.Clone()
		cfgWeapon.Model3DEntry = cfg.Model3D.Weapon
		cfgWeapon.Model3D = nil
		weapon = NewVertices3DEntry(cfgWeapon, materials)
		weaponFaces, countW = weapon.volumes[0].GetFaces()
	}

	lowerFaces, countL := lower.volumes[0].GetFaces()
	upperFaces, countU := upper.volumes[0].GetFaces()
	headFaces, countH := head.volumes[0].GetFaces()
	totalFaces := countL + countU + countH + countW

	upperActionMap := make([]int, len(lower.actionNames))
	for idx, name := range lower.GetActions() {
		upperIdx := 0 // default
		if strings.HasPrefix(name, "both_") {
			upperIdx = v3dFindActionIndex(upper, name)
		} else if strings.HasPrefix(name, "legs_") {
			if strings.Contains(name, "idle") || strings.Contains(name, "stand") {
				upperIdx = v3dFindActionIndex(upper, "torso_stand")
			} else {
				upperIdx = v3dFindActionIndex(upper, "torso_stand")
			}
		}
		upperActionMap[idx] = upperIdx
	}

	v := &Vertices3D{
		elements:       []*Vertices3DEntry{lower, upper, head, weapon},
		totalFaces:     totalFaces,
		currentAction:  -1,
		upperActionMap: upperActionMap,
		facesA:         make([]*Face, totalFaces),
		facesB:         make([]*Face, totalFaces),
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
		} else if i < countL+countU+countH {
			src = (*headFaces)[i-countL-countU]
		} else {
			src = (*weaponFaces)[i-countL-countU-countH]
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

// GetVolume returns the Volume associated with the lower component of the Vertices3D object.
func (v *Vertices3D) GetVolume() *Volume {
	return v.elements[LowerIdx].viewVolume
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
	// idx is the index from lower's actions (since we passed lower's ActionDefinitions to doCreate)
	v.elements[LowerIdx].SetAction(idx)
	if idx >= 0 && idx < len(v.upperActionMap) {
		v.elements[UpperIdx].SetAction(v.upperActionMap[idx])
	} else {
		v.elements[UpperIdx].SetAction(0)
	}
	v.elements[HeadIdx].SetAction(0)
	if v.elements[WeaponIdx] != nil {
		v.elements[WeaponIdx].SetAction(0)
	}
}

// GetDisplacement retrieves the 3D displacement components (dx, dy, dz) from the lower Vertices3DEntry instance.
func (v *Vertices3D) GetDisplacement() (float64, float64, float64) {
	return v.elements[LowerIdx].GetDisplacement()
}

// GetRenderMode retrieves the render mode value associated with the Vertices3D instance by proxying to its lower entry.
func (v *Vertices3D) GetRenderMode() float64 {
	return v.elements[LowerIdx].GetRenderMode()
}

// SetThing assigns the specified IThing instance to all associated Vertices3DEntry components.
func (v *Vertices3D) SetThing(t IThing) {
	for _, e := range v.elements {
		if e != nil {
			e.SetThing(t)
		}
	}
}

// GetVertices retrieves vertex data for rendering at a given tick, including lower, upper, head, and optional weapon parts.
func (v *Vertices3D) GetVertices(tick uint64) (*[]*Face, int, *[]*Face, int, float64, float64) {
	facesL_A, countL, facesL_B, _, lerpT, renderMode := v.elements[LowerIdx].GetVertices(tick)
	facesU_A, countU, facesU_B, _, _, _ := v.elements[UpperIdx].GetVertices(tick)
	facesH_A, countH, facesH_B, _, _, _ := v.elements[HeadIdx].GetVertices(tick)

	// The actual frames were just calculated inside lower.GetVertices, upper.GetVertices, etc.
	// We can retrieve them directly:
	volL_A, volL_B := v.elements[LowerIdx].GetVolumesAt(tick)
	volU_A, volU_B := v.elements[UpperIdx].GetVolumesAt(tick)
	_, _ = v.elements[HeadIdx].GetVolumesAt(tick) // Execute to keep state synced but discard volumes

	tagTorsoA, _ := volL_A.GetVertexTag("tag_torso")
	tagTorsoB, _ := volL_B.GetVertexTag("tag_torso")

	tagHeadA, _ := volU_A.GetVertexTag("tag_head")
	tagHeadB, _ := volU_B.GetVertexTag("tag_head")

	combinedHeadA := geometry.XYZ{X: tagTorsoA.X + tagHeadA.X, Y: tagTorsoA.Y + tagHeadA.Y, Z: tagTorsoA.Z + tagHeadA.Z}
	combinedHeadB := geometry.XYZ{X: tagTorsoB.X + tagHeadB.X, Y: tagTorsoB.Y + tagHeadB.Y, Z: tagTorsoB.Z + tagHeadB.Z}

	for i := 0; i < countL; i++ {
		v3dCopyFacePoints(v.facesA[i], (*facesL_A)[i])
		v3dCopyFacePoints(v.facesB[i], (*facesL_B)[i])
	}

	for i := 0; i < countU; i++ {
		v3dTransformPoints(v.facesA[countL+i], (*facesU_A)[i], tagTorsoA)
		v3dTransformPoints(v.facesB[countL+i], (*facesU_B)[i], tagTorsoB)
	}

	for i := 0; i < countH; i++ {
		v3dTransformPoints(v.facesA[countL+countU+i], (*facesH_A)[i], combinedHeadA)
		v3dTransformPoints(v.facesB[countL+countU+i], (*facesH_B)[i], combinedHeadB)
	}

	if v.elements[WeaponIdx] != nil {
		facesW_A, countW, facesW_B, _, _, _ := v.elements[WeaponIdx].GetVertices(tick)
		_, _ = v.elements[WeaponIdx].GetVolumesAt(tick)

		tagWeaponA, _ := volU_A.GetVertexTag("tag_weapon")
		tagWeaponB, _ := volU_B.GetVertexTag("tag_weapon")

		combinedWeaponA := geometry.XYZ{X: tagTorsoA.X + tagWeaponA.X, Y: tagTorsoA.Y + tagWeaponA.Y, Z: tagTorsoA.Z + tagWeaponA.Z}
		combinedWeaponB := geometry.XYZ{X: tagTorsoB.X + tagWeaponB.X, Y: tagTorsoB.Y + tagWeaponB.Y, Z: tagTorsoB.Z + tagWeaponB.Z}

		for i := 0; i < countW; i++ {
			v3dTransformPoints(v.facesA[countL+countU+countH+i], (*facesW_A)[i], combinedWeaponA)
			v3dTransformPoints(v.facesB[countL+countU+countH+i], (*facesW_B)[i], combinedWeaponB)
		}
	}

	return v.facesAPtr, v.totalFaces, v.facesBPtr, v.totalFaces, lerpT, renderMode
}

// v3dFindActionIndex retrieves the index of the specified action name from a Vertices3DEntry instance, checking for fallbacks.
func v3dFindActionIndex(md1 *Vertices3DEntry, name1 string) int {
	nameLower := strings.ToLower(name1)
	if n, ok := md1.FindActionIndex(nameLower); ok {
		return n
	}
	// Fallback se fallisce (es. se BOTH_DEATH1 manca e c'è TORSO_DEATH1)
	if strings.HasPrefix(nameLower, "both_") {
		torsoName := strings.Replace(nameLower, "both_", "torso_", 1)
		if n, ok := md1.FindActionIndex(torsoName); ok {
			return n
		}
	}
	return 0
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
