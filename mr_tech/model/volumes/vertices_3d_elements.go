package volumes

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

// VirtualVolume represents a virtual 3D volume used for physics locomotion of an articulated owner.
type VirtualVolume struct {
	solver   *collisions.Solver
	elements *Vertices3DElements
}

// NewVirtualVolume creates a virtual volume using the provided vertex elements and rigid body solver.
func NewVirtualVolume(elements *Vertices3DElements, solver *collisions.Solver) *VirtualVolume {
	return &VirtualVolume{
		elements: elements,
		solver:   solver,
	}
}

// Setup initializes the virtual volume's collision solver with object.
func (v *VirtualVolume) Setup(object collisions.IObject) error {
	return v.solver.Setup(object)
}

// GetSolver returns the collision solver associated with the virtual volume.
func (v *VirtualVolume) GetSolver() *collisions.Solver {
	return v.solver
}

// QueryOverlaps queries face overlaps across all active part volumes of Vertices3D.
func (v *VirtualVolume) QueryOverlaps(object aabb.IObject, callback func(object aabb.IObject) bool) {
	for _, el := range v.elements.entries {
		el.comp.QueryOverlaps(object, callback)
	}
}

// Vertices3DElements manages per-part 3D entries, per-frame active volumes, and owner locomotion solver.
type Vertices3DElements struct {
	entries    []*Vertices3DEntry
	vVolume    *VirtualVolume
	volsA      []*Volume
	volsB      []*Volume
	lerpT      float64
	ent        *rigid_body.Entity
	emptyFaces *Faces
}

// NewVertices3DElements builds centered per-part entries and the owner's locomotion VirtualVolume.
func NewVertices3DElements(cfg *config.Thing, materials *materials.Materials) *Vertices3DElements {
	v := &Vertices3DElements{
		emptyFaces: NewFaces(0, false),
	}

	numParts := len(cfg.Model3D.Parts)
	if numParts == 0 {
		panic(fmt.Sprintf("no parts found in vertices_3d"))
	}
	v.entries = make([]*Vertices3DEntry, numParts)
	v.volsA = make([]*Volume, numParts)
	v.volsB = make([]*Volume, numParts)

	salt := uint64(rand.Intn(512))
	restitution := math.Max(cfg.Restitution, 0.20)
	friction := math.Max(cfg.Friction, 0.55)
	gForce := cfg.GForce
	if gForce <= 0 {
		gForce = 9.8 * 50.0
	}

	for i, part := range cfg.Model3D.Parts {
		if part == nil {
			panic(fmt.Sprintf("nil part found in vertices_3d"))
		}
		c := cfg.Clone()
		c.Model3DEntry = part
		c.Model3D = nil
		entry := NewVertices3DEntry(c, materials)
		entry.salt = salt
		partMass := cfg.Mass
		if i > 0 {
			partMass = math.Max(cfg.Mass*0.25, 10.0)
		}
		entry.CenterFrames(partMass, restitution, friction, gForce)
		v.entries[i] = entry
		if len(entry.volumes) == 0 || entry.volumes[0] == nil {
			panic(fmt.Sprintf("empty volumes found in vertices_3d"))
		}
	}

	solver := collisions.NewSolver(true, cfg.Mass, cfg.Restitution, cfg.Friction, cfg.RestitutionCombine, cfg.FrictionCombine, cfg.GForce)
	v.vVolume = NewVirtualVolume(v, solver)
	v.ent = v.vVolume.GetSolver().GetEntity()
	return v
}

// Setup binds the owner locomotion virtual volume solver to thing.
func (v *Vertices3DElements) Setup(thing IThing) error {
	return v.vVolume.Setup(thing)
}

// Bind advances animation frames for entry i at tick and caches its active volumes and lerpT.
func (v *Vertices3DElements) Bind(i int, tick uint64) {
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
func (v *Vertices3DElements) Len() int {
	return len(v.entries)
}

// GetSolver returns the collision solver associated with the virtual volume.
func (v *Vertices3DElements) GetSolver() *collisions.Solver {
	return v.vVolume.GetSolver()
}

// GetEntry returns the vertices entry at index i, or nil if i is out of range.
func (v *Vertices3DElements) GetEntry(i int) *Vertices3DEntry {
	if i >= 0 && i < len(v.entries) {
		return v.entries[i]
	}
	return nil
}

// UpdateEntityBounds updates the owner locomotion entity dimensions from compiled model bounds.
func (v *Vertices3DElements) UpdateEntityBounds(minX, minY, minZ, maxX, maxY, maxZ float64) {
	if minX <= maxX {
		v.ent.SetSize(maxX-minX, maxY-minY, maxZ-minZ)
	}
}
