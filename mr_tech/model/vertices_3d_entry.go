package model

import (
	"fmt"
	"math"
	"math/rand"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/textures"
)

type VAction struct {
	Name  string
	Start int
	End   int
	Clamp bool
}

func NewVAction(name string, start int, end int) *VAction {
	return &VAction{
		Name:  name,
		Start: start,
		End:   end,
		Clamp: false,
	}
}

// Vertices3DEntry represents a structured collection of 3D model data, including frames, actions, and volume association.
type Vertices3DEntry struct {
	viewVolume    *Volume
	volumes       []*Volume
	startFrame    int
	endFrame      int
	startTick     uint64
	currentAction int
	actions       []*VAction
	clampAnim     bool
	tagMap        map[string]int
	volA          *Volume
	volB          *Volume
	lerpT         float64
	singleFrame   bool
	lastVolA      *Volume
	groupSize     float64
	salt          uint64
}

// NewVertices3DEntry creates a new Vertices3DEntry instance with frames, actions, and volume based on the provided configuration.
func NewVertices3DEntry(cfg *config.Thing, materials *Materials) *Vertices3DEntry {
	model := cfg.Model3DEntry
	if len(model.Frames) <= 0 {
		panic(fmt.Sprintf("no Model3DEntry frames for thing %s", cfg.Id))
	}
	if len(model.ActionDefinitions) != len(model.ActionIntervals) {
		panic("action definitions and intervals must be the same length")
	}
	if model.GroupSize <= 0 {
		panic("groupSize must be greater than 0")
	}

	actions := make([]*VAction, len(model.ActionDefinitions))
	for idx, name := range model.ActionDefinitions {
		nameC := v3dCleanString(name)
		actions[idx] = NewVAction(nameC, model.ActionIntervals[idx][0], model.ActionIntervals[idx][1])
		for _, z := range model.ActionClamp {
			if strings.Contains(nameC, v3dCleanString(z)) {
				actions[idx].Clamp = true
				break
			}
		}
	}

	tagCounter := 0
	tagMap := make(map[string]int)
	for _, cfgFrame := range model.Frames {
		for tagName := range cfgFrame.Tags {
			if _, exists := tagMap[tagName]; !exists {
				tagMap[tagName] = tagCounter
				tagCounter++
			}
		}
	}

	volumes := make([]*Volume, len(model.Frames))
	for frameIdx, cfgFrame := range model.Frames {
		baseId := fmt.Sprintf("%s_md1_frame_%d", cfg.Id, frameIdx)
		volume := NewVolume(frameIdx, baseId, "thing", cfg.Mass, cfg.Restitution, cfg.Friction, cfg.GForce)
		for triIdx, tri := range cfgFrame.Triangles {
			label := fmt.Sprintf("%s_%d", baseId, triIdx)
			points := [3]geometry.XYZ{tri.Vertices[0].Pos, tri.Vertices[1].Pos, tri.Vertices[2].Pos}
			var material *textures.Material
			if tri.Material != nil {
				material = materials.GetMaterial(tri.Material)
			}
			face := NewFace(points, label, material)
			face.SetUV(float64(tri.Vertices[0].U), float64(tri.Vertices[0].V), float64(tri.Vertices[1].U), float64(tri.Vertices[1].V), float64(tri.Vertices[2].U), float64(tri.Vertices[2].V))
			face.LockUV(true)
			volume.AddFace(face)
		}
		// Copy tags to volume
		for tagName, tagData := range cfgFrame.Tags {
			volume.SetVertexTag(tagMap[tagName], tagData.Origin, tagData.Axis)
		}
		volume.Rebuild()
		volumes[frameIdx] = volume
	}

	v := &Vertices3DEntry{
		volumes:       volumes,
		groupSize:     float64(model.GroupSize),
		actions:       actions,
		startFrame:    0,
		endFrame:      len(model.Frames) - 1,
		currentAction: -1,
		tagMap:        tagMap,
		volA:          nil,
		volB:          nil,
		lerpT:         0.0,
		lastVolA:      nil,
		salt:          uint64(rand.Intn(len(model.Frames))),
	}

	if len(v.actions) > 0 {
		v.SetAction(0)
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
	if idx < 0 || idx >= len(v.actions) {
		return
	}
	if v.currentAction == idx {
		return
	}
	v.currentAction = idx
	action := v.actions[idx]
	v.startTick = textures.GlobalTick()
	v.startFrame = action.Start
	v.endFrame = action.End
	v.clampAnim = action.Clamp
}

// CompileVertices selects the active animation frame volumes for tick and stores their interpolation factor.
func (v *Vertices3DEntry) CompileVertices(tick uint64) bool {
	if v.singleFrame {
		return false
	}
	tick += v.salt

	animLength := v.endFrame - v.startFrame + 1
	if animLength <= 0 {
		animLength = 1
	}

	lerpT := 0.0
	var relativeFrameA int
	var relativeFrameB int
	if v.clampAnim {
		frameFloat := 0.0
		if tick > v.startTick {
			frameFloat = float64(tick-v.startTick) / v.groupSize
		}
		lastAnim := animLength - 1
		if relativeFrameA = int(frameFloat); relativeFrameA >= lastAnim {
			v.singleFrame = true
			relativeFrameA = lastAnim
			relativeFrameB = lastAnim
			lerpT = 0.0 // Fermo sull'ultimo frame
		} else {
			lerpT = frameFloat - math.Floor(frameFloat)
			if relativeFrameB = relativeFrameA + 1; relativeFrameB >= lastAnim {
				relativeFrameB = lastAnim
			}
		}
	} else {
		// TickGrouped computes the floating-point frame index by dividing the tick by the group size.
		frameFloat := float64(tick) / v.groupSize
		lerpT = frameFloat - math.Floor(frameFloat)
		relativeFrameA = int(frameFloat) % animLength
		relativeFrameB = (int(frameFloat) + 1) % animLength
	}

	v.volA = v.volumes[v.startFrame+relativeFrameA]
	v.volB = v.volumes[v.startFrame+relativeFrameB]
	v.lerpT = lerpT

	if v.lastVolA != v.volA {
		v.lastVolA = v.volA
		return true
	}

	return true
}

// GetVertices computes and retrieves two animation frames and a lerp factor at the given tick for interpolating vertices.
func (v *Vertices3DEntry) GetVertices() (*Faces, *Faces, float64, float64) {
	return v.volA.GetFaces(), v.volB.GetFaces(), v.lerpT, RenderModeModel3D
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
