package vertices_multi_element_3d

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/ithing"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/model/vertices/vertices_element_3d"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

// Elements manages per-part 3D entries, per-frame active volumes, and the living owner's Volume collider.
type Elements struct {
	entries    []*vertices_element_3d.Element3D
	ownerVol   *volumes.Volume
	volsA      []*volumes.Volume
	volsB      []*volumes.Volume
	lerpT      float64
	ent        *rigid_body.Entity
	emptyFaces *volumes.Faces
}

// NewElements builds centered per-part entries and a real Volume + Solver for owner locomotion.
func NewElements(cfg *config.Thing, materials *materials.Materials) *Elements {
	ownerVol := volumes.NewVolumeDynamic(0, cfg.Id+"_owner", "thing", cfg.Mass, cfg.Restitution, cfg.Friction, cfg.RestitutionCombine, cfg.FrictionCombine, cfg.GForce)
	v := &Elements{
		ownerVol:   ownerVol,
		ent:        ownerVol.GetSolver().GetEntity(),
		emptyFaces: volumes.NewFaces(aabb.NewTree(0, 0.0), false),
	}

	numParts := len(cfg.MultiElement3D.Parts)
	if numParts == 0 {
		panic(fmt.Sprintf("no parts found in vertices_3d"))
	}
	v.entries = make([]*vertices_element_3d.Element3D, numParts)
	v.volsA = make([]*volumes.Volume, numParts)
	v.volsB = make([]*volumes.Volume, numParts)

	salt := uint64(rand.Intn(512))
	restitution := math.Max(cfg.Restitution, 0.20)
	friction := math.Max(cfg.Friction, 0.55)
	gForce := cfg.GForce
	if gForce <= 0 {
		gForce = 9.8 * 50.0
	}

	for i, part := range cfg.MultiElement3D.Parts {
		if part == nil {
			panic(fmt.Sprintf("nil part found in vertices_3d"))
		}
		c := cfg.Clone()
		c.Element3D = part
		c.MultiElement3D = nil
		entry := vertices_element_3d.NewElement3D(c, materials)
		entry.SetSalt(salt)
		partMass := cfg.Mass
		if i > 0 {
			partMass = math.Max(cfg.Mass*0.25, 10.0)
		}
		entry.CenterFrames(partMass, restitution, friction, gForce)
		v.entries[i] = entry
		if entry.GetVolumesLen() == 0 || entry.GetVolume(0) == nil {
			panic(fmt.Sprintf("empty volumes found in vertices_3d"))
		}
	}

	return v
}

// Setup binds the owner locomotion Volume solver to thing.
func (v *Elements) Setup(thing ithing.IThing) error {
	return v.ownerVol.Setup(thing)
}

// QueryOverlaps queries face overlaps against the owner's local-space bounding Volume.
func (v *Elements) QueryOverlaps(object aabb.IObject, callback func(object aabb.IObject) bool) {
	v.ownerVol.QueryOverlaps(object, callback)
}

// Bind advances animation frames for entry i at tick and caches its active volumes and lerpT.
func (v *Elements) Bind(i int, tick uint64) {
	element := v.entries[i]
	element.AdvanceAnimation(tick)
	volA, volB, lerpT := element.GetVolumes()
	if i == 0 {
		v.lerpT = lerpT
	}
	v.volsA[i] = volA
	v.volsB[i] = volB
}

// Len returns the number of 3D vertex elements.
func (v *Elements) Len() int {
	return len(v.entries)
}

// GetSolver returns the collision solver associated with the owner locomotion Volume.
func (v *Elements) GetSolver() *collisions.Solver {
	return v.ownerVol.GetSolver()
}

// GetEntry returns the vertices entry at index i, or nil if i is out of range.
func (v *Elements) GetEntry(i int) *vertices_element_3d.Element3D {
	if i >= 0 && i < len(v.entries) {
		return v.entries[i]
	}
	return nil
}

// UpdateEntityBounds rebuilds the owner locomotion box Volume and entity dimensions from compiled model bounds.
func (v *Elements) UpdateEntityBounds(minX, minY, minZ, maxX, maxY, maxZ float64) {
	if minX > maxX {
		return
	}
	v000 := geometry.XYZ{X: minX, Y: minY, Z: minZ}
	v100 := geometry.XYZ{X: maxX, Y: minY, Z: minZ}
	v110 := geometry.XYZ{X: maxX, Y: maxY, Z: minZ}
	v010 := geometry.XYZ{X: minX, Y: maxY, Z: minZ}
	v001 := geometry.XYZ{X: minX, Y: minY, Z: maxZ}
	v101 := geometry.XYZ{X: maxX, Y: minY, Z: maxZ}
	v111 := geometry.XYZ{X: maxX, Y: maxY, Z: maxZ}
	v011 := geometry.XYZ{X: minX, Y: maxY, Z: maxZ}

	v.ownerVol.ClearFaces()
	addQuad := func(a, b, c, d geometry.XYZ) {
		v.ownerVol.AddFace(volumes.NewFace([3]geometry.XYZ{a, b, c}, nil))
		v.ownerVol.AddFace(volumes.NewFace([3]geometry.XYZ{a, c, d}, nil))
	}
	addQuad(v000, v010, v110, v100) // bottom
	addQuad(v001, v101, v111, v011) // top
	addQuad(v000, v100, v101, v001) // front
	addQuad(v010, v011, v111, v110) // back
	addQuad(v000, v001, v011, v010) // left
	addQuad(v100, v110, v111, v101) // right
	v.ownerVol.Rebuild()
	v.ent.SetSize(maxX-minX, maxY-minY, maxZ-minZ)
}
