package metrics

import "github.com/markel1974/godoom/mr_tech/model"

// Shadows represents a structure used to manage shadow configurations, including dimensions, matrices, and Flash integration.
type Shadows struct {
	shadowWidth    int32
	shadowHeight   int32
	shadowAspect   float32
	shadowSpace    [16]float32
	shadowProj     [16]float32
	shadowProjPtr  *float32
	shadowSpacePtr *float32

	flash *model.Flash
}

// NewShadows initializes and returns a new Shadows instance configured with the provided Flash instance.
func NewShadows(flash *model.Flash) *Shadows {
	m := &Shadows{
		flash: flash,
	}
	m.shadowProjPtr = &m.shadowProj[0]
	m.shadowSpacePtr = &m.shadowSpace[0]
	m.Rebuild(1024, 1024)
	return m
}

// GetShadowSize returns the shadow map's width and height as two int32 values.
func (m *Shadows) GetShadowSize() (int32, int32) {
	return m.shadowWidth, m.shadowHeight
}

// GetFlashAspect returns the aspect ratio used for the flashlight's projection, derived from the shadow's dimensions.
func (m *Shadows) GetFlashAspect() float32 {
	return m.shadowAspect
}

// Rebuild updates the shadow dimensions, recalculates the shadow aspect ratio, and rebuilds related projection matrices.
func (m *Shadows) Rebuild(width, height int32) {
	m.shadowWidth = width   // * 0.5)
	m.shadowHeight = height // * 0.5)
	m.shadowAspect = float32(m.shadowWidth) / float32(m.shadowHeight)
	if m.shadowAspect == 0 {
		m.shadowAspect = 1.0
	}
	m.flash.Rebuild(ndcRange)
	m.updateShadowProj()
}

// CreateShadowSpace calculates the shadow space matrix for rendering shadow maps using a flashlight position and projection.
// It generates a local shadow view matrix based on the flashlight's LookAt calculation and combines it with the main view matrix.
func (m *Shadows) CreateShadowSpace(mainView [16]float32, flashOffsetX, flashOffsetY float32) {
	// Local ShadowLight Space (LookAt calculation)
	posViewX, posViewY, posViewZ := flashOffsetX, flashOffsetY, float32(0.0)
	targetX, targetY, targetZ := float32(0.0), float32(0.0), -float32(m.flash.GetZFar())
	// Forward, Right, Up for the flashlight
	ffX, ffY, ffZ := normalize(targetX-posViewX, targetY-posViewY, targetZ-posViewZ)
	rrX, rrY, rrZ := normalize(crossProduct(ffX, ffY, ffZ, 0.0, 1.0, 0.0))
	uuX, uuY, uuZ := crossProduct(rrX, rrY, rrZ, ffX, ffY, ffZ)
	// Local Translation
	tLocX := -dotProduct(rrX, rrY, rrZ, posViewX, posViewY, posViewZ)
	tLocY := -dotProduct(uuX, uuY, uuZ, posViewX, posViewY, posViewZ)
	tLocZ := dotProduct(ffX, ffY, ffZ, posViewX, posViewY, posViewZ)
	shadowViewLocal := [16]float32{
		rrX, uuX, -ffX, 0,
		rrY, uuY, -ffY, 0,
		rrZ, uuZ, -ffZ, 0,
		tLocX, tLocY, tLocZ, 1,
	}
	shadowView := MatrixMultiply4x4(shadowViewLocal, mainView)
	shadowSpace := MatrixMultiply4x4(m.shadowProj, shadowView)
	copy(m.shadowSpace[:], shadowSpace[:])
}

func (m *Shadows) GetShadowSpacePtr() *float32 {
	return m.shadowSpacePtr
}

// updateFlashProj computes and updates the flashlight projection matrix using its field of view and near/far plane distances.
func (m *Shadows) updateShadowProj() {
	shadowFov := float32(m.flash.GetShadowFovScale())
	zNearFlash := float32(m.flash.GetZNear())
	zFarFlash := float32(m.flash.GetZFar())

	diffZ := zNearFlash - zFarFlash
	if diffZ == 0 {
		diffZ = 1.0
	}
	shadowProj := [16]float32{
		shadowFov / m.shadowAspect, 0, 0, 0,
		0, shadowFov, 0, 0,
		0, 0, (zFarFlash + zNearFlash) / diffZ, -1,
		0, 0, (2 * zFarFlash * zNearFlash) / diffZ, 0,
	}
	copy(m.shadowProj[:], shadowProj[:])
}
