package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// MSFaces represents a pair of connected Faces within a 3D volume, providing bidirectional linking between structures.
type MSFaces struct {
	face0 *Face
	face1 *Face
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
	x := pos.X - cfg.Radius
	y := pos.Y - cfg.Radius
	z := pos.Z
	w := cfg.Radius * 2
	h := cfg.Radius * 2
	d := cfg.Height

	volume := NewVolumeDetails3d(0, "wax", "thing", x, y, z, w, h, d, cfg.Mass, cfg.Restitution, cfg.Friction, cfg.GForce)
	f := &VerticesMultiSprite{
		volume: volume,
	}
	f.faces = make([]*MSFaces, len(cfg.MultiSprite.Materials))
	for viewIdx, view := range cfg.MultiSprite.Materials {
		if view == nil || len(view.Frames) == 0 {
			continue
		}
		material := materials.GetMaterial(view)
		f0, f1 := f.createFaces(w, h, material)
		f.faces[viewIdx] = &MSFaces{face0: f0, face1: f1}
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
	return f, c, f, c, 0.0
}

/*
// SetViewAngle calculates the relative angle between the camera and the entity and updates the current view angle index.
func (v *VerticesMultiSprite) SetViewAngle(cameraPos, entityPos geometry.XYZ, entityYaw float64) {
	// 1. Calcolo del vettore direzione dalla telecamera all'entità
	dx := cameraPos.X - entityPos.X
	dy := cameraPos.Y - entityPos.Y // Considerando Y come profondità (Z nel tuo CreateCoords)
	// 2. Angolo assoluto tra telecamera ed entità
	angleToCam := math.Atan2(dy, dx)
	// 3. Delta angolare (Angolo Telecamera - Yaw Entità)
	// Normalizziamo l'angolo per assicurarci che sia tra 0 e 2*Pi
	relativeAngle := math.Mod(angleToCam-entityYaw+(math.Pi*2), math.Pi*2)
	sectorMax := len(v.faces)
	sectorSize := (math.Pi * 2) / float64(sectorMax)
	index := int(math.Floor((relativeAngle + (sectorSize / 2.0)) / sectorSize))
	v.currentAngle = index % sectorMax
	v.compute()
}
*/

// SetViewAngle calculates the relative angle between the camera and the entity and updates the current view angle index.
func (v *VerticesMultiSprite) SetViewAngle(cameraPos, entityPos geometry.XYZ, entityYaw float64) {
	dx := cameraPos.X - entityPos.X
	dy := cameraPos.Y - entityPos.Y

	angleToCam := math.Atan2(dy, dx)
	relativeAngle := math.Mod(angleToCam-entityYaw+(math.Pi*2), math.Pi*2)

	sectorSize := (math.Pi * 2) / 32.0
	index := int(math.Floor((relativeAngle + (sectorSize / 2.0)) / sectorSize))

	const viewOffset = 3
	v.currentAngle = (index + viewOffset) % 32

	v.compute()
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
	v.volume.AddFace(v.viewFaces.face0)
	v.volume.AddFace(v.viewFaces.face1)
	v.volume.Rebuild()
}

// createFaces generates two triangular faces based on the given width, height, and material animation.
func (v *VerticesMultiSprite) createFaces(width float64, height float64, anim *textures.Material) (*Face, *Face) {
	// Triangolo 0: Top-Left, Bottom-Left, Bottom-Right
	halfW := width * 0.5
	t0 := [3]geometry.XYZ{
		{X: -halfW, Y: 0.0, Z: height}, // TL
		{X: -halfW, Y: 0.0, Z: 0.0},    // BL
		{X: halfW, Y: 0.0, Z: 0.0},     // BR
	}
	f0 := NewFace(nil, t0, "", anim)
	// Passiamo V=0 per il top e V=-1 per il bottom (diventerà 1 nel renderer)
	f0.SetUV(0.0, 0.0, 0.0, -1.0, 1.0, -1.0)
	f0.LockUV(true)
	// Triangolo 1: Top-Left, Bottom-Right, Top-Right
	t1 := [3]geometry.XYZ{
		{X: -halfW, Y: 0.0, Z: height}, // TL
		{X: halfW, Y: 0.0, Z: 0.0},     // BR
		{X: halfW, Y: 0.0, Z: height},  // TR
	}
	f1 := NewFace(nil, t1, "", anim)
	// TL: (0,0), BR: (1,-1), TR: (1,0)
	f1.SetUV(0.0, 0.0, 1.0, -1.0, 1.0, 0.0)
	f1.LockUV(true)
	return f0, f1
}
