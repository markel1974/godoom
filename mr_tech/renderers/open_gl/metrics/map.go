package metrics

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model"
)

// Map represents settings and transformations for rendering maps, including orthographic size, view parameters, and shadows.
type Map struct {
	orthoSize  float32
	roomZNear  float32
	roomZFar   float32
	lightCamY  float32
	mapCenterX float32
	mapCenterZ float32

	mainView  [16]float32
	roomProj  [16]float32
	roomView  [16]float32
	roomSpace [16]float32

	mainViewPtr  *float32
	roomProjPtr  *float32
	roomViewPtr  *float32
	roomSpacePtr *float32
}

// NewMap initializes and returns a pointer to a Map instance configured using the given Flash object.
func NewMap() *Map {
	m := &Map{}
	m.mainViewPtr = &m.mainView[0]
	m.roomProjPtr = &m.roomProj[0]
	m.roomViewPtr = &m.roomView[0]
	m.roomSpacePtr = &m.roomSpace[0]
	m.SetOrthoSize(float32(640), 0.0, 8192.0)
	m.SetMapCenter(0.0, 0.0, 0.0)
	return m
}

// GetScale2d calculates the horizontal and vertical scales based on the given width and height of the viewport.
func (m *Map) GetScale2d(width, height int32) (float32, float32) {
	aspect := float32(width) / float32(height)
	scaleX := (ndcRange / aspect) * float32(model.HFov)
	scaleY := ndcRange * float32(model.VFov)
	return scaleX, scaleY
}

// GetScale3d calculates and returns the scaling factors for 3D projection based on screen dimensions, aspect ratio, and FOV.
func (m *Map) GetScale3d(width, height int32, aspectRatio float32, fovVerticalDegrees float32) (float32, float32) {
	// Window aspect ratio (e.g. 1280/960 = 1.333)
	aspect := float32(width) / float32(height)
	// Vertical Field of View (75.0 is the Quake/Retro standard for 4:3 screens)
	// On 16:9 screens, if it appears too "zoomed in", increase to 80 or 90.
	//const fovVerticalDegrees = 80.0
	fovYRad := (fovVerticalDegrees * math.Pi) / 180.0
	// Base focal length derived from field of view angle
	focal := float32(1.0 / math.Tan(float64(fovYRad)/2.0))
	// Canonical OpenGL mapping (Perspective Projection)
	// The Y axis uses pure focal length. The X axis is corrected (divided) by the aspect ratio.
	scaleY := focal * aspectRatio
	scaleX := focal / aspect
	return scaleX, scaleY
}

// GetOrthoSize retrieves the orthographic size associated with the current Map instance.
func (m *Map) GetOrthoSize() float32 {
	return m.orthoSize
}

// SetOrthoSize sets the orthographic size and near/far room clipping planes, updating the room projection matrix.
func (m *Map) SetOrthoSize(orthoSize, zNearRoom, zFarRoom float32) {
	m.orthoSize = orthoSize
	m.roomZNear = zNearRoom
	m.roomZFar = zFarRoom
	m.updateRoomProj(m.orthoSize, m.roomZNear, m.roomZFar)
}

// GetRoomZNear returns the near clipping plane distance for the room projection.
func (m *Map) GetRoomZNear() float32 {
	return m.roomZNear
}

// GetRoomZFar retrieves the far clipping distance of the room's projection matrix.
func (m *Map) GetRoomZFar() float32 {
	return m.roomZFar
}

// SetMapCenter updates the map's center coordinates and light camera Y position, recalculating the room view matrix.
func (m *Map) SetMapCenter(cx float32, cz float32, lightCamY float32) {
	m.mapCenterX = cx
	m.lightCamY = lightCamY
	m.mapCenterZ = cz
	m.updateRoomView(m.mapCenterX, m.lightCamY, m.mapCenterZ)
}

// GetLightCamY retrieves the Y-coordinate of the light camera position stored in the Map instance.
func (m *Map) GetLightCamY() float32 {
	return m.lightCamY
}

// GetMapCenterX returns the X-coordinate of the map's center.
func (m *Map) GetMapCenterX() float32 {
	return m.mapCenterX
}

// GetMapCenterZ retrieves the Z-coordinate of the map's center position.
func (m *Map) GetMapCenterZ() float32 {
	return m.mapCenterZ
}

// CreateRoomSpace generates and returns the room space and main view transformation matrices based on the provided view matrix.
func (m *Map) CreateRoomSpace(vi *model.ViewMatrix) {
	// Clean extraction (World Space: Z-UP)
	// Setup Camera (Main View)
	sinA, cosA := vi.GetAngleFull()
	fX, fY, fZ := float32(cosA), float32(0.0), float32(-sinA)
	rX, rY, rZ := -fZ, float32(0.0), fX
	uX, uY, uZ := float32(0.0), float32(1.0), float32(0.0)
	wX, wY, wZ := vi.GetView()
	// Spatial mapping for OpenGL (X, Z, -Y)
	camX, camY, camZ := float32(wX), float32(wZ), float32(-wY)
	mainView := [16]float32{
		rX, uX, -fX, 0,
		rY, uY, -fY, 0,
		rZ, uZ, -fZ, 0,
		-dotProduct(rX, rY, rZ, camX, camY, camZ),
		-dotProduct(uX, uY, uZ, camX, camY, camZ),
		dotProduct(fX, fY, fZ, camX, camY, camZ), 1,
	}
	copy(m.mainView[:], mainView[:])
}

func (m *Map) GetRoomSpacePtr() *float32 {
	return m.roomSpacePtr
}

func (m *Map) GetMainViewPtr() *float32 {
	return m.mainViewPtr
}

func (m *Map) GetMainView() [16]float32 {
	return m.mainView
}

// updateRoomProj updates the room projection matrix and computes the room space matrix based on the given parameters.
func (m *Map) updateRoomProj(orthoSize, zNearRoom, zFarRoom float32) {
	if orthoSize == 0 {
		orthoSize = 1.0
	}
	diffZ := zFarRoom - zNearRoom
	if diffZ == 0 {
		diffZ = 1.0
	}
	roomProj := [16]float32{
		1.0 / orthoSize, 0, 0, 0,
		0, 1.0 / orthoSize, 0, 0,
		0, 0, -ndcRange / diffZ, 0,
		0, 0, -(zFarRoom + zNearRoom) / diffZ, 1,
	}
	copy(m.roomProj[:], roomProj[:])
	roomSpace := MatrixMultiply4x4(m.roomProj, m.roomView)
	copy(m.roomSpace[:], roomSpace[:])
}

// updateRoomView updates the room view matrix and recalculates the combined room space matrix.
func (m *Map) updateRoomView(lX, lY, lZ float32) {
	const skew = 0.02
	roomView := [16]float32{
		1, 0, 0, 0,
		skew, skew, 1, 0,
		0, -1, 0, 0,
		-lX, lY, -lZ, 1,
	}
	copy(m.roomView[:], roomView[:])
	roomSpace := MatrixMultiply4x4(m.roomProj, m.roomView)
	copy(m.roomSpace[:], roomSpace[:])
}

/*
// CreateSpaces2d generates transformation matrices for 2D projections, including room space, flashlight space, and main view.
func (m *MapMetrics) CreateSpaces2d(vi *model.ViewMatrix, flashOffsetX, flashOffsetY float32) ([16]float32, [16]float32, [16]float32) {
	// Clean extraction (World Space: Z-UP)
	// Setup Camera (Main View)
	sinA, cosA := vi.GetAngleFull()
	fX, fY, fZ := float32(cosA), float32(0.0), float32(-sinA)
	rX, rY, rZ := -fZ, float32(0.0), fX
	uX, uY, uZ := float32(0.0), float32(1.0), float32(0.0)
	wX, wY, wZ := vi.GetView()
	// Spatial mapping for OpenGL (X, Z, -Y)
	camX, camY, camZ := float32(wX), float32(wZ), float32(-wY)
	mainView := [16]float32{
		rX, uX, -fX, 0,
		rY, uY, -fY, 0,
		rZ, uZ, -fZ, 0,
		-dot(rX, rY, rZ, camX, camY, camZ),
		-dot(uX, uY, uZ, camX, camY, camZ),
		dot(fX, fY, fZ, camX, camY, camZ), 1,
	}
	// Local ShadowLight Space (LookAt calculation)
	pitchShear := float32(-vi.GetPitch())
	flashDirY := pitchShear / (ndcRange * float32(model.VFov))
	posViewX, posViewY, posViewZ := flashOffsetX, flashOffsetY, float32(0.0)
	targetX, targetY, targetZ := float32(0.0), flashDirY*float32(m.flash.GetZFar()), -float32(m.flash.GetZFar())
	// Forward, Right, Up for the flashlight
	ffX, ffY, ffZ := normalize(targetX-posViewX, targetY-posViewY, targetZ-posViewZ)
	rrX, rrY, rrZ := normalize(cross(ffX, ffY, ffZ, 0.0, 1.0, 0.0))
	uuX, uuY, uuZ := cross(rrX, rrY, rrZ, ffX, ffY, ffZ) // Already normalized
	// Local Translation
	tLocX := -dot(rrX, rrY, rrZ, posViewX, posViewY, posViewZ)
	tLocY := -dot(uuX, uuY, uuZ, posViewX, posViewY, posViewZ)
	tLocZ := dot(ffX, ffY, ffZ, posViewX, posViewY, posViewZ)
	flashViewLocal := [16]float32{
		rrX, uuX, -ffX, 0,
		rrY, uuY, -ffY, 0,
		rrZ, uuZ, -ffZ, 0,
		tLocX, tLocY, tLocZ, 1,
	}
	// Final Matrices
	flashView := MatrixMultiply4x4(flashViewLocal, mainView)
	flashSpace := MatrixMultiply4x4(m.flashProj, flashView)
	return m.roomSpace, flashSpace, mainView
}
*/

// GetFovScaleFactor retrieves the scaling factor applied to the field of view (FOV) for perspective calculations.
//func (m *MapMetrics) GetFovScaleFactor() float32 {
//	return m.fovScaleFactor
//}

// SetFovScaleFactor sets the field-of-view (FOV) scale factor and updates the scaled FOV value.
//func (m *MapMetrics) SetFovScaleFactor(value float32) {
//	m.fovScaleFactor = value
//	m.scaleFovY = m.fovScaleFactor * float32(model.VFov)
//}
