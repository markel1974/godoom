package volumes

import (
	"fmt"
	"math"
	"math/rand"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// VAction defines a named animation action with frame bounds and optional clamping behavior.
type VAction struct {
	Name  string
	Start int
	End   int
	Clamp bool
}

// NewVAction creates a vertex action with the given name and frame interval, with clamping disabled.
func NewVAction(name string, start int, end int) *VAction {
	return &VAction{
		Name:  name,
		Start: start,
		End:   end,
		Clamp: false,
	}
}

// Vertices3DEntry stores 3D volume animation state, frame ranges, actions, and interpolation data.
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
	tick          uint64
}

// NewVertices3DEntry builds a vertex-based 3D entry from thing configuration, frames, tags, actions, and materials.
// It panics if required frame, action interval, or group size configuration is invalid.
func NewVertices3DEntry(thing IThing, cfg *config.Thing, materials *materials.Materials) *Vertices3DEntry {
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

	vols := make([]*Volume, len(model.Frames))
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
		vols[frameIdx] = volume
	}

	v := &Vertices3DEntry{
		volumes:       vols,
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
		v.VerticesSetAction(0)
	}

	if v.startFrame == v.endFrame {
		v.singleFrame = true
		v.volA = v.volumes[v.startFrame]
		v.volB = v.volumes[v.startFrame]
		v.lerpT = 0.0
	}

	v.viewVolume = v.volumes[0]

	for _, f := range v.volumes {
		f.SetThing(thing)
	}

	return v
}

// GetAABB returns the axis-aligned bounding box of the entry's entity.
func (v *Vertices3DEntry) GetAABB() *aabb.AABB {
	return v.viewVolume.GetEntity().GetAABB()
}

// VerticesGetVolume returns the current view volume for this vertices entry.
func (v *Vertices3DEntry) VerticesGetVolume() *Volume {
	return v.viewVolume
}

// VerticesGetEntity returns the physics entity associated with the current view volume.
func (v *Vertices3DEntry) VerticesGetEntity() *physics.Entity {
	return v.viewVolume.GetEntity()
}

// VerticesSetAction selects the animation action at idx and resets timing for its frame interval.
func (v *Vertices3DEntry) VerticesSetAction(idx int) {
	if idx < 0 || idx >= len(v.actions) {
		return
	}
	if v.currentAction == idx {
		return
	}
	v.currentAction = idx
	action := v.actions[idx]
	v.startTick = v.tick
	v.startFrame = action.Start
	v.endFrame = action.End
	v.clampAnim = action.Clamp
}

// VerticesCompile updates the active animation frames and interpolation factor for the given tick.
func (v *Vertices3DEntry) VerticesCompile(tick uint64) bool {
	if v.singleFrame {
		return false
	}
	v.tick = tick + v.salt

	animLength := v.endFrame - v.startFrame + 1
	if animLength <= 0 {
		animLength = 1
	}

	lerpT := 0.0
	var relativeFrameA int
	var relativeFrameB int
	if v.clampAnim {
		frameFloat := 0.0
		if v.tick > v.startTick {
			frameFloat = float64(v.tick-v.startTick) / v.groupSize
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
		frameFloat := float64(v.tick) / v.groupSize
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

// VerticesGet returns the two face sets, interpolation factor, and model 3D render mode.
func (v *Vertices3DEntry) VerticesGet() (*Faces, *Faces, float64, float64) {
	return v.volA.GetFaces(), v.volB.GetFaces(), v.lerpT, RenderModeModel3D
}

// GetVolumes returns the current source and target volumes with the interpolation factor between them.
func (v *Vertices3DEntry) GetVolumes() (*Volume, *Volume, float64) {
	return v.volA, v.volB, v.lerpT
}

// VerticesGetCenter returns the x, y, and z coordinates of the view volume entity's center.
func (v *Vertices3DEntry) VerticesGetCenter() (float64, float64, float64) {
	return v.viewVolume.GetEntity().GetCenter()
}

// GetTagIndex returns the index for the named tag, or -1 if the tag is not found.
func (v *Vertices3DEntry) GetTagIndex(name string) int {
	if idx, ok := v.tagMap[name]; ok {
		return idx
	}
	fmt.Printf("[warning] Vertices3DEntry %s: tag not found\n", name)
	return -1
}

// v3dCleanString normalizes s by trimming surrounding whitespace and converting it to lowercase.
func v3dCleanString(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}
