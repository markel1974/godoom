package model

import (
	"fmt"
	"math"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// VerticesMD1 represents a structured collection of 3D model data, including frames, actions, and volume association.
type VerticesMD1 struct {
	viewVolume      *Volume
	volumes         []*Volume
	startFrame      int
	endFrame        int
	startTick       uint64
	currentAction   int
	actionIntervals [][2]int
	actionNames     []string
	actionNamesC    map[string]int
	actionClamps    []bool
	clampAnim       bool
}

// NewVerticesMD1 creates a new VerticesMD1 instance with frames, actions, and volume based on the provided configuration.
func NewVerticesMD1(cfg *config.Thing, materials *Materials) *VerticesMD1 {
	if len(cfg.MD1.Frames) == 0 {
		panic(fmt.Sprintf("no MD1 frames for thing %s", cfg.Id))
	}

	actionClamps := make([]bool, len(cfg.MD1.ActionDefinitions))
	actionNames := make([]string, len(cfg.MD1.ActionDefinitions))
	actionNamesC := make(map[string]int)

	for idx, name := range cfg.MD1.ActionDefinitions {
		nameC := md1CleanString(name)
		actionNames[idx] = nameC
		actionNamesC[nameC] = idx
		for _, z := range cfg.MD1.ActionClamp {
			if strings.Contains(nameC, md1CleanString(z)) {
				actionClamps[idx] = true
				break
			}
		}
	}

	v := &VerticesMD1{
		volumes:         make([]*Volume, len(cfg.MD1.Frames)),
		actionIntervals: cfg.MD1.ActionIntervals,
		actionNames:     actionNames,
		actionNamesC:    actionNamesC,
		actionClamps:    actionClamps,
		startFrame:      0,
		endFrame:        len(cfg.MD1.Frames) - 1,
		currentAction:   -1,
	}
	if v.endFrame < 0 {
		v.endFrame = 0
	}
	if len(v.actionIntervals) > 0 {
		v.SetAction(0)
	}
	//entity := physics.NewEntity(x, y, z, w, h, d, cfg.Mass, cfg.Restitution, cfg.Friction, cfg.GForce)
	for frameIdx, cfgFrame := range cfg.MD1.Frames {
		baseId := fmt.Sprintf("%s_md1_frame_%d", cfg.Id, frameIdx)
		volume := NewVolume(frameIdx, baseId, "thing", cfg.Mass, cfg.Restitution, cfg.Friction, cfg.GForce)
		for triIdx, tri := range cfgFrame.Triangles {
			tag := fmt.Sprintf("%s_%d", baseId, triIdx)
			points := [3]geometry.XYZ{tri.Vertices[0].Pos, tri.Vertices[1].Pos, tri.Vertices[2].Pos}
			material := materials.GetMaterial(tri.Material)
			face := NewFace(points, tag, material)
			face.SetUV(float64(tri.Vertices[0].U), float64(tri.Vertices[0].V), float64(tri.Vertices[1].U), float64(tri.Vertices[1].V), float64(tri.Vertices[2].U), float64(tri.Vertices[2].V))
			face.LockUV(true)
			volume.AddFace(face)
		}
		// Copy tags to volume
		for tagName, tagVec := range cfgFrame.Tags {
			volume.SetVertexTag(tagVec, tagName)
		}
		volume.Rebuild()
		v.volumes[frameIdx] = volume
	}
	v.viewVolume = v.volumes[0]
	//v.rootEntity = v.viewVolume.GetEntity()
	return v
}

// GetVolume retrieves the Volume instance associated with the VertexMD2.
func (v *VerticesMD1) GetVolume() *Volume {
	return v.viewVolume
}

// GetActions returns the list of action names associated with the VerticesMD1 instance.
func (v *VerticesMD1) GetActions() []string {
	return v.actionNames
}

// GetEntity returns the physics.Entity instance associated with the VerticesMD1 viewVolume.
func (v *VerticesMD1) GetEntity() *physics.Entity {
	return v.viewVolume.GetEntity()
}

// GetAABB returns the axis-aligned bounding box (AABB) of the associated entity in the view volume.
func (v *VerticesMD1) GetAABB() *physics.AABB {
	return v.viewVolume.GetEntity().GetAABB()
}

// SetAction updates the start and end frame of the VertexMD2 based on the action index provided.
func (v *VerticesMD1) SetAction(idx int) {
	if idx < 0 || idx >= len(v.actionIntervals) {
		return
	}
	if v.currentAction == idx {
		return
	}
	v.currentAction = idx
	v.startFrame = v.actionIntervals[idx][0]
	v.endFrame = v.actionIntervals[idx][1]
	v.startTick = textures.GlobalTick()
	v.clampAnim = false
	if idx < len(v.actionClamps) {
		v.clampAnim = v.actionClamps[idx]
	}
}

// FindActionIndex searches for the index of the given action name in the actionNames list, ignoring case and returning success status.
func (v *VerticesMD1) FindActionIndex(name string) (int, bool) {
	nameLower := md1CleanString(name)
	n, k := v.actionNamesC[nameLower]
	return n, k
}

// GetVertices computes and retrieves two animation frames and a lerp factor at the given tick for interpolating vertices.
func (v *VerticesMD1) GetVertices(tick uint64) (*[]*Face, int, *[]*Face, int, float64, float64) {
	// Se non ci sono frame, restituisce vuoto
	if len(v.volumes) == 0 {
		return nil, 0, nil, 0, 0.0, v.GetBillboard()
	}
	// Se c'è un solo frame nell'animazione, restituisce lo stesso frame due volte senza lerp
	if v.startFrame == v.endFrame {
		s := v.volumes[v.startFrame]
		faces, faceCount := s.GetFaces()
		return faces, faceCount, faces, faceCount, 0.0, v.GetBillboard()
	}
	const groupSize = 6.0
	var frameFloat float64
	if v.clampAnim {
		elapsed := uint64(0)
		if tick > v.startTick {
			elapsed = tick - v.startTick
		}
		frameFloat = float64(elapsed) / groupSize
	} else {
		frameFloat = textures.TickGrouped(tick, int(groupSize))
	}

	// Calcoliamo la durata dell'animazione corrente in termini di numero di frame
	animLength := v.endFrame - v.startFrame + 1
	if animLength <= 0 {
		animLength = 1
	}

	// Troviamo l'indice relativo all'interno dell'animazione corrente
	relativeFrameA := int(frameFloat)
	relativeFrameB := relativeFrameA + 1
	lerpT := frameFloat - math.Floor(frameFloat)

	if v.clampAnim {
		if relativeFrameA >= animLength-1 {
			relativeFrameA = animLength - 1
			relativeFrameB = animLength - 1
			lerpT = 0.0 // Fermo sull'ultimo frame
		} else if relativeFrameB >= animLength-1 {
			relativeFrameB = animLength - 1
		}
	} else {
		// Looping animation
		relativeFrameA = relativeFrameA % animLength
		relativeFrameB = relativeFrameB % animLength
	}

	idxA := v.startFrame + relativeFrameA
	idxB := v.startFrame + relativeFrameB

	curr := v.volumes[idxA]
	next := v.volumes[idxB]

	facesA, faceCountA := curr.GetFaces()
	facesB, faceCountB := next.GetFaces()

	return facesA, faceCountA, facesB, faceCountB, lerpT, v.GetBillboard()
}

// GetVolumesAt calculates and returns the volumes for a specific tick, without mutating state.
func (v *VerticesMD1) GetVolumesAt(tick uint64) (*Volume, *Volume) {
	if len(v.volumes) == 0 {
		return nil, nil
	}
	if v.startFrame == v.endFrame {
		return v.volumes[v.startFrame], v.volumes[v.startFrame]
	}
	const groupSize = 6.0
	var frameFloat float64
	if v.clampAnim {
		elapsed := uint64(0)
		if tick > v.startTick {
			elapsed = tick - v.startTick
		}
		frameFloat = float64(elapsed) / groupSize
	} else {
		frameFloat = textures.TickGrouped(tick, int(groupSize))
	}

	animLength := v.endFrame - v.startFrame + 1
	if animLength <= 0 {
		animLength = 1
	}

	relativeFrameA := int(frameFloat)
	relativeFrameB := relativeFrameA + 1

	if v.clampAnim {
		if relativeFrameA >= animLength-1 {
			relativeFrameA = animLength - 1
			relativeFrameB = animLength - 1
		} else if relativeFrameB >= animLength-1 {
			relativeFrameB = animLength - 1
		}
	} else {
		relativeFrameA = relativeFrameA % animLength
		relativeFrameB = relativeFrameB % animLength
	}

	return v.volumes[v.startFrame+relativeFrameA], v.volumes[v.startFrame+relativeFrameB]
}

// GetDisplacement retrieves the displacement vector (dx, dy, dz) by getting the center position of the associated entity.
func (v *VerticesMD1) GetDisplacement() (float64, float64, float64) {
	return v.viewVolume.GetEntity().GetCenter()
}

// GetBillboard returns a constant value, typically used to represent the billboard distance for the VerticesMD1 instance.
func (v *VerticesMD1) GetBillboard() float64 {
	return 2.0
}

// SetThing sets the IThing instance associated with the VerticesMD1 volume.
func (v *VerticesMD1) SetThing(t IThing) {
	for _, f := range v.volumes {
		f.SetThing(t)
	}
}

// md1CleanString normalizes a string by converting it to lowercase and trimming leading and trailing whitespace.
func md1CleanString(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}
