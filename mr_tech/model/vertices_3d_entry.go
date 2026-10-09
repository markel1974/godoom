package model

import (
	"fmt"
	"math"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Vertices3DEntry represents a structured collection of 3D model data, including frames, actions, and volume association.
type Vertices3DEntry struct {
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
	tagMap          map[string]int
}

// NewVertices3DEntry creates a new Vertices3DEntry instance with frames, actions, and volume based on the provided configuration.
func NewVertices3DEntry(cfg *config.Thing, materials *Materials) *Vertices3DEntry {
	if len(cfg.Model3DEntry.Frames) == 0 {
		panic(fmt.Sprintf("no Model3DEntry frames for thing %s", cfg.Id))
	}

	actionClamps := make([]bool, len(cfg.Model3DEntry.ActionDefinitions))
	actionNames := make([]string, len(cfg.Model3DEntry.ActionDefinitions))
	actionNamesC := make(map[string]int)

	for idx, name := range cfg.Model3DEntry.ActionDefinitions {
		nameC := v3dCleanString(name)
		actionNames[idx] = nameC
		actionNamesC[nameC] = idx
		for _, z := range cfg.Model3DEntry.ActionClamp {
			if strings.Contains(nameC, v3dCleanString(z)) {
				actionClamps[idx] = true
				break
			}
		}
	}

	v := &Vertices3DEntry{
		volumes:         make([]*Volume, len(cfg.Model3DEntry.Frames)),
		actionIntervals: cfg.Model3DEntry.ActionIntervals,
		actionNames:     actionNames,
		actionNamesC:    actionNamesC,
		actionClamps:    actionClamps,
		startFrame:      0,
		endFrame:        len(cfg.Model3DEntry.Frames) - 1,
		currentAction:   -1,
		tagMap:          make(map[string]int),
	}

	tagCounter := 0
	for _, cfgFrame := range cfg.Model3DEntry.Frames {
		for tagName := range cfgFrame.Tags {
			if _, exists := v.tagMap[tagName]; !exists {
				v.tagMap[tagName] = tagCounter
				tagCounter++
			}
		}
	}
	if v.endFrame < 0 {
		v.endFrame = 0
	}
	if len(v.actionIntervals) > 0 {
		v.SetAction(0)
	}
	//entity := physics.NewEntity(x, y, z, w, h, d, cfg.Mass, cfg.Restitution, cfg.Friction, cfg.GForce)
	for frameIdx, cfgFrame := range cfg.Model3DEntry.Frames {
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
			volume.SetVertexTag(v.tagMap[tagName], tagVec)
		}
		volume.Rebuild()
		v.volumes[frameIdx] = volume
	}
	v.viewVolume = v.volumes[0]
	//v.rootEntity = v.viewVolume.GetEntity()
	return v
}

// GetVolume retrieves the Volume instance associated with the VertexMD2.
func (v *Vertices3DEntry) GetVolume() *Volume {
	return v.viewVolume
}

// GetActions returns the list of action names associated with the Vertices3DEntry instance.
func (v *Vertices3DEntry) GetActions() []string {
	return v.actionNames
}

// GetEntity returns the physics.Entity instance associated with the Vertices3DEntry viewVolume.
func (v *Vertices3DEntry) GetEntity() *physics.Entity {
	return v.viewVolume.GetEntity()
}

// GetAABB returns the axis-aligned bounding box (AABB) of the associated entity in the view volume.
func (v *Vertices3DEntry) GetAABB() *aabb.AABB {
	return v.viewVolume.GetEntity().GetAABB()
}

// SetAction updates the start and end frame of the VertexMD2 based on the action index provided.
func (v *Vertices3DEntry) SetAction(idx int) {
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
func (v *Vertices3DEntry) FindActionIndex(name string) (int, bool) {
	nameLower := v3dCleanString(name)
	n, k := v.actionNamesC[nameLower]
	return n, k
}

// GetVertices computes and retrieves two animation frames and a lerp factor at the given tick for interpolating vertices.
func (v *Vertices3DEntry) GetVertices(tick uint64) (*[]*Face, int, *[]*Face, int, float64, float64) {
	volA, volB, lerpT := v.GetVolumesAt(tick)
	if volA == nil || volB == nil {
		return nil, 0, nil, 0, 0.0, v.GetRenderMode()
	}
	facesA, count := volA.GetFaces()
	facesB, _ := volB.GetFaces()
	return facesA, count, facesB, count, lerpT, v.GetRenderMode()
}

// GetVolumesAt retrieves the precise start and end frames for interpolation at the given tick.
func (v *Vertices3DEntry) GetVolumesAt(tick uint64) (*Volume, *Volume, float64) {
	if len(v.volumes) == 0 {
		return nil, nil, 0.0
	}
	if v.startFrame == v.endFrame {
		return v.volumes[v.startFrame], v.volumes[v.startFrame], 0.0
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

	frameA := v.startFrame + relativeFrameA
	frameB := v.startFrame + relativeFrameB

	return v.volumes[frameA], v.volumes[frameB], lerpT
}

// GetDisplacement retrieves the displacement vector (dx, dy, dz) by getting the center position of the associated entity.
func (v *Vertices3DEntry) GetDisplacement() (float64, float64, float64) {
	return v.viewVolume.GetEntity().GetCenter()
}

// GetRenderMode returns a constant value, typically used to represent the renderMode distance for the Vertices3DEntry instance.
func (v *Vertices3DEntry) GetRenderMode() float64 {
	return RenderModeModel3D
}

// SetThing sets the IThing instance associated with the Vertices3DEntry volume.
func (v *Vertices3DEntry) SetThing(t IThing) {
	for _, f := range v.volumes {
		f.SetThing(t)
	}
}

// GetTagIndex returns the internal index of a tag by its name, or -1 if not found.
func (v *Vertices3DEntry) GetTagIndex(name string) int {
	if idx, ok := v.tagMap[name]; ok {
		return idx
	}
	return -1
}

// v3dCleanString normalizes a string by converting it to lowercase and trimming leading and trailing whitespace.
func v3dCleanString(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}
