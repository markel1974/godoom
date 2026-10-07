package metrics

import "github.com/markel1974/godoom/mr_tech/model"

// Shadows represents a structure for handling shadow-related transformations and projections in a rendering system.
type Shadows struct {
	shadowWidth        int32
	shadowHeight       int32
	shadowAspect       float32
	shadowSpace        [16]float32
	shadowProj         [16]float32
	shadowView         [16]float32
	shadowViewLocal    [16]float32
	shadowProjPtr      *float32
	shadowSpacePtr     *float32
	shadowViewPtr      *float32
	shadowViewLocalPtr *float32

	flash *model.Flash
}

// NewShadows creates and initializes a new Shadows instance with a given Flash object and default shadow settings.
func NewShadows(flash *model.Flash) *Shadows {
	m := &Shadows{
		flash: flash,
	}
	m.shadowProjPtr = &m.shadowProj[0]
	m.shadowSpacePtr = &m.shadowSpace[0]
	m.shadowViewPtr = &m.shadowView[0]
	m.shadowViewLocalPtr = &m.shadowViewLocal[0]
	m.Rebuild(1024, 1024)
	return m
}

// GetShadowSize retrieves the width and height of the shadow map as two int32 values.
func (m *Shadows) GetShadowSize() (int32, int32) {
	return m.shadowWidth, m.shadowHeight
}

// GetFlashAspect returns the aspect ratio of the shadow, calculated as the ratio of shadow width to shadow height.
func (m *Shadows) GetFlashAspect() float32 {
	return m.shadowAspect
}

// Rebuild updates shadow dimensions, calculates shadow aspect ratio, and refreshes shadow projection matrices.
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

// CreateShadowSpace computes the shadow-space matrix by integrating view and projection matrices for shadow mapping.
func (m *Shadows) CreateShadowSpace(mainView *float32, flashOffsetX, flashOffsetY float32) {
	// Local ShadowLight Space (LookAt calculation)
	posViewX, posViewY, posViewZ := flashOffsetX, flashOffsetY, float32(0.0)
	targetX, targetY, targetZ := float32(0.0), float32(0.0), -float32(m.flash.GetZFar())

	ffX, ffY, ffZ := normalize(targetX-posViewX, targetY-posViewY, targetZ-posViewZ)
	rrX, rrY, rrZ := normalize(crossProduct(ffX, ffY, ffZ, 0.0, 1.0, 0.0))
	uuX, uuY, uuZ := crossProduct(rrX, rrY, rrZ, ffX, ffY, ffZ)

	tLocX := -dotProduct(rrX, rrY, rrZ, posViewX, posViewY, posViewZ)
	tLocY := -dotProduct(uuX, uuY, uuZ, posViewX, posViewY, posViewZ)
	tLocZ := dotProduct(ffX, ffY, ffZ, posViewX, posViewY, posViewZ)

	m.shadowViewLocal[0] = rrX
	m.shadowViewLocal[1] = uuX
	m.shadowViewLocal[2] = -ffX
	m.shadowViewLocal[3] = 0
	m.shadowViewLocal[4] = rrY
	m.shadowViewLocal[5] = uuY
	m.shadowViewLocal[6] = -ffY
	m.shadowViewLocal[7] = 0
	m.shadowViewLocal[8] = rrZ
	m.shadowViewLocal[9] = uuZ
	m.shadowViewLocal[10] = -ffZ
	m.shadowViewLocal[11] = 0
	m.shadowViewLocal[12] = tLocX
	m.shadowViewLocal[13] = tLocY
	m.shadowViewLocal[14] = tLocZ
	m.shadowViewLocal[15] = 1

	MatrixMultiply4x4Ptr(m.shadowViewPtr, m.shadowViewLocalPtr, mainView)
	MatrixMultiply4x4Ptr(m.shadowSpacePtr, m.shadowProjPtr, m.shadowViewPtr)
}

// GetShadowSpacePtr returns a pointer to the shadow space data buffer used for shadow matrix transformations.
func (m *Shadows) GetShadowSpacePtr() *float32 {
	return m.shadowSpacePtr
}

// updateShadowProj updates the shadow projection matrix based on the flashlight's field of view, aspect ratio, and z-planes.
func (m *Shadows) updateShadowProj() {
	shadowFov := float32(m.flash.GetShadowFovScale())
	zNearFlash := float32(m.flash.GetZNear())
	zFarFlash := float32(m.flash.GetZFar())
	diffZ := zNearFlash - zFarFlash
	if diffZ == 0 {
		diffZ = 1.0
	}
	m.shadowProj[0] = shadowFov / m.shadowAspect
	m.shadowProj[1] = 0
	m.shadowProj[2] = 0
	m.shadowProj[3] = 0
	m.shadowProj[4] = 0
	m.shadowProj[5] = shadowFov
	m.shadowProj[6] = 0
	m.shadowProj[7] = 0
	m.shadowProj[8] = 0
	m.shadowProj[9] = 0
	m.shadowProj[10] = (zFarFlash + zNearFlash) / diffZ
	m.shadowProj[11] = -1
	m.shadowProj[12] = 0
	m.shadowProj[13] = 0
	m.shadowProj[14] = (2 * zFarFlash * zNearFlash) / diffZ
	m.shadowProj[15] = 0
}
