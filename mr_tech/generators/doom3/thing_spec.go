package doom3

import (
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/doom3/proc"
)

// ThingSpec holds the default physical and rendering configuration for a Doom 3 entity type.
type ThingSpec struct {
	proc.ResolvedEntity
	Name  string
	Kind  config.ThingType
	Speed float64
}

func NewThingSpec() *ThingSpec {
	return &ThingSpec{}
}

func (t *ThingSpec) Setup(classname string, spawnClass string) {
	prefix := strings.Split(strings.ToLower(classname), "_")[0]
	t.Name = classname
	t.Kind = config.ThingItemDef
	t.Speed = 0.0
	t.Health = 100.0
	t.Mass = 15.0
	if prefix == "monster" || strings.Contains(spawnClass, "idai") {
		t.Kind = config.ThingEnemyDef
		t.Speed = 1150.0
		t.Mass = 35.0
	}
}

func (t *ThingSpec) Prepare(spec *ConfigSpec) {
	t.ModelPath = spec.ModelPath
	t.IdleAnim = spec.IdleAnim
	t.WalkAnim = spec.WalkAnim
	t.RunAnim = spec.RunAnim
	t.AttackAnim = spec.AttackAnim
	t.PainAnim = spec.PainAnim
	t.DeathAnim = spec.DeathAnim
	t.HeadMesh = spec.HeadMesh
	t.HeadJoint = spec.HeadJoint
	t.Ragdoll = spec.Ragdoll
	t.AFBodies = spec.AFBodies
	t.Kind = spec.Kind
	t.Mass = spec.Mass
	t.Speed = spec.Speed
	t.Health = spec.Health
}

func (t *ThingSpec) Update(spec *proc.ResolvedEntity, bodies []*proc.AFBody) {
	if len(spec.ModelPath) > 0 {
		t.ModelPath = spec.ModelPath
	}
	if len(spec.HeadMesh) > 0 {
		t.HeadMesh = spec.HeadMesh
	}
	if len(spec.HeadJoint) > 0 {
		t.HeadJoint = spec.HeadJoint
	}
	if len(spec.IdleAnim) > 0 {
		t.IdleAnim = spec.IdleAnim
	}
	if len(spec.WalkAnim) > 0 {
		t.WalkAnim = spec.WalkAnim
	}
	if len(spec.RunAnim) > 0 {
		t.RunAnim = spec.RunAnim
	}
	if len(spec.AttackAnim) > 0 {
		t.AttackAnim = spec.AttackAnim
	}
	if len(spec.PainAnim) > 0 {
		t.PainAnim = spec.PainAnim
	}
	if len(spec.DeathAnim) > 0 {
		t.DeathAnim = spec.DeathAnim
	}
	if len(spec.Ragdoll) > 0 {
		t.Ragdoll = spec.Ragdoll
	}
	if len(spec.AFBodies) > 0 {
		t.AFBodies = spec.AFBodies
	}
	if spec.Mass > 0 {
		t.Mass = spec.Mass
	}
	if spec.Health > 0 {
		t.Health = spec.Health
	}
	if len(bodies) > 0 {
		t.AFBodies = bodies
	}
}

// ResolveEntity reconstructs the Doom3ThingSpec for a classname by combining static mappings with
// dynamic .def inheritance and ReadDirFilter discovery on the PK4 archive.
func (t *ThingSpec) ResolveEntity(classname string, defs *proc.Defs) bool {
	resolvedSpec, err := defs.ResolveEntity(classname)
	if err != nil {
		return false
	}
	spawnClass := strings.ToLower(resolvedSpec.SpawnClass)
	t.Setup(classname, spawnClass)
	if configSpec, ok := GetEntitySpec(classname); ok {
		t.Prepare(configSpec)
	}
	bodies, _ := defs.ResolveAFBodies(classname)
	t.Update(resolvedSpec, bodies)
	return true
}
