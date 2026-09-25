package model

import (
	"fmt"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
)

// VerticesMD3 represents a higher-level structure containing multiple VerticesMD1 instances and related face data.
type VerticesMD3 struct {
	lower  *VerticesMD1
	upper  *VerticesMD1
	head   *VerticesMD1
	weapon *VerticesMD1

	facesA    []*Face
	facesB    []*Face
	facesAPtr *[]*Face
	facesBPtr *[]*Face

	totalFaces    int
	entity        *physics.Entity
	currentAction int
}

// NewVerticesMD3 initializes and returns a new instance of VerticesMD3 based on the given configuration and materials.
// It creates vertices for the lower, upper, head, and optionally a weapon, combining their faces into a unified structure.
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

	var weapon *VerticesMD1
	var countW int
	var weaponFaces *[]*Face
	if cfg.MD3.Weapon != nil {
		cfgWeapon := cfg.Clone()
		cfgWeapon.MD1 = cfg.MD3.Weapon
		cfgWeapon.MD3 = nil
		weapon = NewVerticesMD2(cfgWeapon, materials)
		weaponFaces, countW = weapon.volumes[0].GetFaces()
	}

	lowerFaces, countL := lower.volumes[0].GetFaces()
	upperFaces, countU := upper.volumes[0].GetFaces()
	headFaces, countH := head.volumes[0].GetFaces()
	totalFaces := countL + countU + countH + countW

	v := &VerticesMD3{
		lower:         lower,
		upper:         upper,
		head:          head,
		weapon:        weapon,
		totalFaces:    totalFaces,
		currentAction: -1,
		facesA:        make([]*Face, totalFaces),
		facesB:        make([]*Face, totalFaces),
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

// GetVolume retrieves the Volume instance associated with the lower VerticesMD1 of the VerticesMD3.
func (v *VerticesMD3) GetVolume() *Volume {
	return v.lower.viewVolume
}

// GetEntity retrieves the physics.Entity instance associated with the VerticesMD3 object.
func (v *VerticesMD3) GetEntity() *physics.Entity {
	return v.entity
}

// GetAABB retrieves the axis-aligned bounding box (AABB) associated with the entity of the VerticesMD3 instance.
func (v *VerticesMD3) GetAABB() *physics.AABB {
	return v.entity.GetAABB()
}

// SetAction updates the actions for the lower, upper, head, and weapon vertices based on the provided index.
func (v *VerticesMD3) SetAction(idx int) {
	if v.currentAction == idx {
		return
	}
	v.currentAction = idx
	// idx is the index from lower's actions (since we passed lower's ActionDefinitions to doCreate)
	v.lower.SetAction(idx)

	actionName := v.lower.GetActionName(idx)
	if actionName == "" {
		return
	}

	// Map LEGS_ or BOTH_ actions to TORSO_ actions
	// If it's BOTH_, both lower and upper should have it.
	// If it's LEGS_RUN, we want TORSO_STAND or similar.
	// We can use a simple mapping here:
	if strings.HasPrefix(actionName, "BOTH_") {
		v.upper.SetActionByName(actionName)
	} else if strings.HasPrefix(actionName, "LEGS_") {
		// Just a default fallback: while legs are moving, torso stands.
		// If the enemy shoots, Enemy logic would need a way to set torso action independently.
		// For now, we enforce a default torso state so it doesn't stay stuck in BOTH_DEATH1.
		if strings.Contains(actionName, "IDLE") || strings.Contains(actionName, "STAND") {
			v.upper.SetActionByName("TORSO_STAND")
		} else {
			v.upper.SetActionByName("TORSO_STAND") // Could be TORSO_STAND2, etc.
		}
	}

	v.head.SetAction(0)
	if v.weapon != nil {
		v.weapon.SetAction(0)
	}
}

// GetDisplacement returns the displacement vector (dx, dy, dz) from the lower part of the model in the 3D space.
func (v *VerticesMD3) GetDisplacement() (float64, float64, float64) {
	return v.lower.GetDisplacement()
}

// GetBillboard retrieves the billboard distance value from the underlying lower VerticesMD1 instance.
func (v *VerticesMD3) GetBillboard() float64 {
	return v.lower.GetBillboard()
}

// SetThing assigns the specified IThing instance to all components of VerticesMD3, including lower, upper, head, and weapon.
func (v *VerticesMD3) SetThing(t IThing) {
	v.lower.SetThing(t)
	v.upper.SetThing(t)
	v.head.SetThing(t)
	if v.weapon != nil {
		v.weapon.SetThing(t)
	}
}

// transformPoints updates the destination face points by translating source face points by the given origin offset.
func transformPoints(dst *Face, src *Face, origin geometry.XYZ) {
	pts := src.GetPoints()
	dst.tri[0] = geometry.XYZ{X: pts[0].X + origin.X, Y: pts[0].Y + origin.Y, Z: pts[0].Z + origin.Z}
	dst.tri[1] = geometry.XYZ{X: pts[1].X + origin.X, Y: pts[1].Y + origin.Y, Z: pts[1].Z + origin.Z}
	dst.tri[2] = geometry.XYZ{X: pts[2].X + origin.X, Y: pts[2].Y + origin.Y, Z: pts[2].Z + origin.Z}
}

// copyFacePoints copies the vertex points from the source Face to the destination Face.
func copyFacePoints(dst *Face, src *Face) {
	pts := src.GetPoints()
	dst.tri[0] = pts[0]
	dst.tri[1] = pts[1]
	dst.tri[2] = pts[2]
}

// GetVertices retrieves and transforms the vertex data for the current tick, including interpolation and hierarchical adjustments.
func (v *VerticesMD3) GetVertices(tick uint64) (*[]*Face, int, *[]*Face, int, float64, float64) {
	facesL_A, countL, facesL_B, _, lerpT, billboard := v.lower.GetVertices(tick)
	facesU_A, countU, facesU_B, _, _, _ := v.upper.GetVertices(tick)
	facesH_A, countH, facesH_B, _, _, _ := v.head.GetVertices(tick)

	// The actual frames were just calculated inside lower.GetVertices, upper.GetVertices, etc.
	// We can retrieve them directly:
	volL_A, volL_B := v.lower.GetVolumesAt(tick)
	volU_A, volU_B := v.upper.GetVolumesAt(tick)
	_, _ = v.head.GetVolumesAt(tick) // Execute to keep state synced but discard volumes

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

	if v.weapon != nil {
		facesW_A, countW, facesW_B, _, _, _ := v.weapon.GetVertices(tick)
		_, _ = v.weapon.GetVolumesAt(tick)

		tagWeaponA := volU_A.Tags["tag_weapon"]
		tagWeaponB := volU_B.Tags["tag_weapon"]

		combinedWeaponA := geometry.XYZ{X: tagTorsoA.X + tagWeaponA.X, Y: tagTorsoA.Y + tagWeaponA.Y, Z: tagTorsoA.Z + tagWeaponA.Z}
		combinedWeaponB := geometry.XYZ{X: tagTorsoB.X + tagWeaponB.X, Y: tagTorsoB.Y + tagWeaponB.Y, Z: tagTorsoB.Z + tagWeaponB.Z}

		for i := 0; i < countW; i++ {
			transformPoints(v.facesA[countL+countU+countH+i], (*facesW_A)[i], combinedWeaponA)
			transformPoints(v.facesB[countL+countU+countH+i], (*facesW_B)[i], combinedWeaponB)
		}
	}

	return v.facesAPtr, v.totalFaces, v.facesBPtr, v.totalFaces, lerpT, billboard
}
