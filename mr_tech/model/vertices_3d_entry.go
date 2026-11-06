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
	actionClamps    []bool
	clampAnim       bool
	tagMap          map[string]int
	volA            *Volume
	volB            *Volume
	lerpT           float64
	singleFrame     bool
	relativeFrameA  int
	groupSize       float64
}

// NewVertices3DEntry creates a new Vertices3DEntry instance with frames, actions, and volume based on the provided configuration.
func NewVertices3DEntry(cfg *config.Thing, materials *Materials) *Vertices3DEntry {
	if len(cfg.Model3DEntry.Frames) == 0 {
		panic(fmt.Sprintf("no Model3DEntry frames for thing %s", cfg.Id))
	}

	actionClamps := make([]bool, len(cfg.Model3DEntry.ActionDefinitions))
	actionNames := make([]string, len(cfg.Model3DEntry.ActionDefinitions))
	//actionNamesC := make(map[string]int)

	for idx, name := range cfg.Model3DEntry.ActionDefinitions {
		nameC := v3dCleanString(name)
		actionNames[idx] = nameC
		//actionNamesC[nameC] = idx
		for _, z := range cfg.Model3DEntry.ActionClamp {
			if strings.Contains(nameC, v3dCleanString(z)) {
				actionClamps[idx] = true
				break
			}
		}
	}

	v := &Vertices3DEntry{
		volumes:         make([]*Volume, len(cfg.Model3DEntry.Frames)),
		groupSize:       float64(cfg.Model3DEntry.GroupSize),
		actionIntervals: cfg.Model3DEntry.ActionIntervals,
		actionNames:     actionNames,
		//actionNamesC:    actionNamesC,
		actionClamps:   actionClamps,
		startFrame:     0,
		endFrame:       len(cfg.Model3DEntry.Frames) - 1,
		currentAction:  -1,
		tagMap:         make(map[string]int),
		volA:           nil,
		volB:           nil,
		lerpT:          0.0,
		relativeFrameA: -1,
	}

	if v.groupSize <= 0 {
		panic("groupSize must be greater than 0")
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
			var material *textures.Material
			if tri.Material != nil {
				material = materials.GetMaterial(tri.Material)
			}
			face := NewFace(points, tag, material)
			face.SetUV(float64(tri.Vertices[0].U), float64(tri.Vertices[0].V), float64(tri.Vertices[1].U), float64(tri.Vertices[1].V), float64(tri.Vertices[2].U), float64(tri.Vertices[2].V))
			face.LockUV(true)
			volume.AddFace(face)
		}
		// Copy tags to volume
		for tagName, tagData := range cfgFrame.Tags {
			volume.SetVertexTag(v.tagMap[tagName], tagData.Origin, tagData.Axis)
		}
		volume.Rebuild()
		v.volumes[frameIdx] = volume
	}

	if v.startFrame == v.endFrame {
		v.singleFrame = true
		v.volA = v.volumes[v.startFrame]
		v.volB = v.volumes[v.startFrame]
		v.lerpT = 0.0
	}

	v.viewVolume = v.volumes[0]
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

// CompileVertices selects the active animation frame volumes for tick and stores their interpolation factor.
func (v *Vertices3DEntry) CompileVertices(tick uint64) bool {
	if v.singleFrame {
		return false
	}

	var frameFloat float64
	if v.clampAnim {
		if tick > v.startTick {
			frameFloat = float64(tick-v.startTick) / v.groupSize
		} else {
			frameFloat = 0.0
		}
	} else {
		// TickGrouped computes the floating-point frame index by dividing the tick by the group size.
		frameFloat = float64(tick) / v.groupSize
	}
	relativeFrameA := int(frameFloat)
	relativeFrameB := relativeFrameA + 1
	animLength := v.endFrame - v.startFrame + 1
	if animLength <= 0 {
		animLength = 1
	}
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
	v.volA = v.volumes[v.startFrame+relativeFrameA]
	v.volB = v.volumes[v.startFrame+relativeFrameB]
	v.lerpT = lerpT

	if relativeFrameA != v.relativeFrameA {
		v.relativeFrameA = relativeFrameA
		return true
	}

	return false
}

// GetVertices computes and retrieves two animation frames and a lerp factor at the given tick for interpolating vertices.
func (v *Vertices3DEntry) GetVertices() (*Faces, *Faces, float64, float64) {
	if v.volA == nil || v.volB == nil {
		return nil, nil, 0.0, v.GetRenderMode()
	}
	facesA := v.volA.GetFaces()
	facesB := v.volB.GetFaces()
	return facesA, facesB, v.lerpT, v.GetRenderMode()
}

// GetVolumes retrieves the precise start and end frames for interpolation at the given tick.
func (v *Vertices3DEntry) GetVolumes() (*Volume, *Volume, float64) {
	return v.volA, v.volB, v.lerpT
}

// GetVerticesCenter retrieves the displacement vector (dx, dy, dz) by getting the center position of the associated entity.
func (v *Vertices3DEntry) GetVerticesCenter() (float64, float64, float64) {
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
	fmt.Printf("[warning] Vertices3DEntry %s: tag not found\n", name)
	return -1
}

// v3dCleanString normalizes a string by converting it to lowercase and trimming leading and trailing whitespace.
func v3dCleanString(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}
