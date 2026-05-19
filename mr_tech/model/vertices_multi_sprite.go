package model

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// MSFaces represents a pair of connected Faces within a 3D volume, providing bidirectional linking between structures.
type MSFaces struct {
	faces []*Face
}

// VerticesMultiSprite represents a 3D entity composed of MSFaces and organized within a Volume.
type VerticesMultiSprite struct {
	volume        *Volume
	baseTexName   string
	currentAction int
	currentAngle  int
	faces         []*MSFaces
	viewFaces     *MSFaces
}

// NewVerticesMultiSprite creates a new VerticesMultiSprite instance with geometry, physics, and animation information, based on input config.
func NewVerticesMultiSprite(cfg *config.Thing, pos geometry.XYZ, materials *Materials) *VerticesMultiSprite {
	w := cfg.Radius * 2
	h := cfg.Radius * 2

	volume := NewVolume(0, "wax", "thing", cfg.Mass, cfg.Restitution, cfg.Friction, cfg.GForce)
	f := &VerticesMultiSprite{
		volume: volume,
	}
	f.faces = make([]*MSFaces, len(cfg.MultiSprite.Materials))
	for viewIdx, view := range cfg.MultiSprite.Materials {
		if view == nil || len(view.Frames) == 0 {
			continue
		}
		material := materials.GetMaterial(view)
		faces := f.createFaces(w, h, material)
		f.faces[viewIdx] = &MSFaces{faces: faces}
	}
	f.compute()
	return f
}

// GetVolume returns the pointer to the Volume instance associated with the VerticesMultiSprite object.
func (v *VerticesMultiSprite) GetVolume() *Volume {
	return v.volume
}

// GetVertices retrieves the faces and associated data for the current frame and returns them with a default displacement value.
func (v *VerticesMultiSprite) GetVertices(tick uint64) ([]*Face, int, []*Face, int, float64) {
	f, c := v.volume.GetFaces()
	return f[:], c, f[:], c, 0.0
}

// SetAction updates the current action index if the provided index is within bounds and triggers a recomputation of vertices.
func (v *VerticesMultiSprite) SetAction(idx int) {
	if idx < 0 || idx >= len(v.faces) {
		return
	}
	v.currentAction = idx
	v.compute()
}

// GetDisplacement retrieves the displacement coordinates (X, Y, Z) of the volume's bottom-left position.
func (v *VerticesMultiSprite) GetDisplacement() (float64, float64, float64) {
	return v.volume.entity.GetBottomLeft()
}

// GetBillboard returns the billboard orientation value for the VerticesMultiSprite instance.
func (v *VerticesMultiSprite) GetBillboard() float64 {
	return 1.0
}

// SetThing assigns an IThing instance to the internal volume of the VerticesMultiSprite object.
func (v *VerticesMultiSprite) SetThing(t IThing) {
	v.volume.SetThing(t)
}

// compute updates the current view faces and rebuilds the volume geometry based on the active view angle.
func (v *VerticesMultiSprite) compute() {
	viewFaces := v.faces[v.currentAngle]
	if viewFaces == v.viewFaces {
		return
	}
	if v.viewFaces = viewFaces; v.viewFaces == nil {
		if v.viewFaces = v.faces[0]; v.viewFaces == nil {
			return
		}
	}
	v.volume.ClearFaces()
	for _, f := range v.viewFaces.faces {
		v.volume.AddFace(f)
	}
	v.volume.Rebuild()
}

// createFaces generates a 3D box geometry (12 triangular faces) based on the given width, height, and material animation.
func (v *VerticesMultiSprite) createFaces(width float64, height float64, material *textures.Material) []*Face {
	if material != nil {
		tex := material.CurrentFrame()
		if tex != nil {
			texW, texH := tex.Size()
			scaleW, scaleH := tex.GetScaleFactor()
			width = float64(texW) * scaleW
			height = float64(texH) * scaleH
		}
	}

	halfW := width * 0.5
	halfY := halfW // Profondità dell'AABB fisico

	// Vertici del Box 3D (Fisica)
	v000 := geometry.XYZ{X: -halfW, Y: -halfY, Z: 0.0}
	v100 := geometry.XYZ{X: halfW, Y: -halfY, Z: 0.0}
	v110 := geometry.XYZ{X: halfW, Y: halfY, Z: 0.0}
	v010 := geometry.XYZ{X: -halfW, Y: halfY, Z: 0.0}

	v001 := geometry.XYZ{X: -halfW, Y: -halfY, Z: height}
	v101 := geometry.XYZ{X: halfW, Y: -halfY, Z: height}
	v111 := geometry.XYZ{X: halfW, Y: halfY, Z: height}
	v011 := geometry.XYZ{X: -halfW, Y: halfY, Z: height}

	var faces []*Face

	// 1. MURI FISICI INVISIBILI (NIL MATERIAL)
	// Queste facce servono solo al solver e all'AABB tree.
	addPhysicsQuad := func(vA, vB, vC, vD geometry.XYZ) {
		f0 := NewFace([3]geometry.XYZ{vA, vB, vC}, "", nil) // NIL MATERIAL
		f1 := NewFace([3]geometry.XYZ{vA, vC, vD}, "", nil) // NIL MATERIAL
		faces = append(faces, f0, f1)
	}

	addPhysicsQuad(v001, v000, v100, v101) // Front
	addPhysicsQuad(v111, v110, v010, v011) // Back
	addPhysicsQuad(v011, v010, v000, v001) // Left
	addPhysicsQuad(v101, v100, v110, v111) // Right
	addPhysicsQuad(v011, v001, v101, v111) // Top
	addPhysicsQuad(v000, v010, v110, v100) // Bottom

	// 2. PIANO VISIVO (Render)
	// Posizionato ESATTAMENTE al centro (Y=0) con il materiale visibile.
	t0 := [3]geometry.XYZ{
		{X: -halfW, Y: 0.0, Z: height}, // TL
		{X: -halfW, Y: 0.0, Z: 0.0},    // BL
		{X: halfW, Y: 0.0, Z: 0.0},     // BR
	}
	f0 := NewFace(t0, "", material)
	f0.SetUV(0.0, 0.0, 0.0, -1.0, 1.0, -1.0)
	f0.LockUV(true)

	t1 := [3]geometry.XYZ{
		{X: -halfW, Y: 0.0, Z: height}, // TL
		{X: halfW, Y: 0.0, Z: 0.0},     // BR
		{X: halfW, Y: 0.0, Z: height},  // TR
	}
	f1 := NewFace(t1, "", material)
	f1.SetUV(0.0, 0.0, 1.0, -1.0, 1.0, 0.0)
	f1.LockUV(true)

	faces = append(faces, f0, f1)

	return faces
}
