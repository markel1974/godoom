package proc

import (
	"strconv"
	"strings"
)

// ResolvedEntity holds the fully resolved model, head, animation, and physical dependencies for a Doom 3 entity.
type ResolvedEntity struct {
	Classname  string
	SpawnClass string
	ModelPath  string
	HeadMesh   string
	HeadJoint  string
	Ragdoll    string
	AFBodies   []*AFBody
	IdleAnim   string
	WalkAnim   string
	RunAnim    string
	AttackAnim string
	PainAnim   string
	DeathAnim  string
	Mass       float64
	Health     float64
	Anims      map[string]string
}

func NewResolvedEntity(cn string) *ResolvedEntity {
	return &ResolvedEntity{
		Classname: cn,
		Anims:     make(map[string]string),
	}
}

func (res *ResolvedEntity) Resolve(def *DefData, af *AFData) error {
	if sc, ok := def.GetEntityProperty(res.Classname, "spawnclass"); ok {
		res.SpawnClass = sc
	}
	if hStr, ok := def.GetEntityProperty(res.Classname, "health"); ok {
		if h, err := strconv.ParseFloat(hStr, 64); err == nil {
			res.Health = h
		}
	}
	if mStr, ok := def.GetEntityProperty(res.Classname, "mass"); ok {
		if m, err := strconv.ParseFloat(mStr, 64); err == nil {
			res.Mass = m
		}
	}

	modelProp, _ := def.GetEntityProperty(res.Classname, "model")
	modelPropClean := cleanAssetPath(modelProp)

	if strings.HasSuffix(modelPropClean, ".md5mesh") || strings.HasSuffix(modelPropClean, ".lwo") {
		res.ModelPath = modelPropClean
	} else if len(modelPropClean) > 0 {
		if mesh, mAnims, ok := def.ResolveModel(modelPropClean); ok {
			res.ModelPath = mesh
			res.Anims = mAnims
		}
	}

	// Resolve attached head ("def_head" -> entityDef/model -> mesh)
	if headDef, ok := def.GetEntityProperty(res.Classname, "def_head"); ok && len(headDef) > 0 {
		headClean := cleanAssetPath(headDef)
		if strings.HasSuffix(headClean, ".md5mesh") {
			res.HeadMesh = headClean
		} else if headMesh, _, hOk := def.ResolveModel(headClean); hOk && len(headMesh) > 0 {
			res.HeadMesh = headMesh
		} else if headModelProp, hmOk := def.GetEntityProperty(headClean, "model"); hmOk {
			hmClean := cleanAssetPath(headModelProp)
			if strings.HasSuffix(hmClean, ".md5mesh") {
				res.HeadMesh = hmClean
			} else if hMesh, _, mOk := def.ResolveModel(hmClean); mOk {
				res.HeadMesh = hMesh
			}
		}
	}
	if hj, ok := def.GetEntityProperty(res.Classname, "head_joint"); ok && len(hj) > 0 {
		res.HeadJoint = hj
	}
	if rag, ok := def.GetEntityProperty(res.Classname, "ragdoll"); ok && len(rag) > 0 {
		res.Ragdoll = rag
		if bodies, bOk := af.ResolveBodies(rag); bOk {
			res.AFBodies = bodies
		}
	} else if bodies, bOk := af.ResolveBodies(res.Classname); bOk {
		res.Ragdoll = res.Classname
		res.AFBodies = bodies
	}
	return nil
}
