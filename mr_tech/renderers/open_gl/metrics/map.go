package metrics

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model"
)

// Map represents a data structure used to manage orthographic settings, transformations, and camera projections.
type Map struct {
	orthoSize  float32
	roomZNear  float32
	roomZFar   float32
	lightCamY  float32
	mapCenterX float32
	mapCenterZ float32

	view       [16]float32
	proj       [16]float32
	invView    [16]float32
	viewPtr    *float32
	projPtr    *float32
	invViewPtr *float32

	mainView     [16]float32
	roomProj     [16]float32
	roomView     [16]float32
	roomSpace    [16]float32
	mainViewPtr  *float32
	roomProjPtr  *float32
	roomViewPtr  *float32
	roomSpacePtr *float32
}

// NewMap creates and initializes a new Map instance with default settings and returns a pointer to it.
func NewMap() *Map {
	m := &Map{}
	m.viewPtr = &m.view[0]
	m.projPtr = &m.proj[0]
	m.invViewPtr = &m.invView[0]

	m.mainViewPtr = &m.mainView[0]
	m.roomProjPtr = &m.roomProj[0]
	m.roomViewPtr = &m.roomView[0]
	m.roomSpacePtr = &m.roomSpace[0]
	m.SetOrthoSize(float32(640), 0.0, 8192.0)
	m.SetMapCenter(0.0, 0.0, 0.0)
	return m
}

// GetScale2d calculates and returns the horizontal and vertical scale factors for a 2D scene based on window dimensions.
func (m *Map) GetScale2d(width, height int32) (float32, float32) {
	aspect := float32(width) / float32(height)
	scaleX := (ndcRange / aspect) * float32(model.HFov)
	scaleY := ndcRange * float32(model.VFov)
	return scaleX, scaleY
}

// GetScale3d calculates and returns the 3D scaling factors along the X and Y axes based on aspect ratio and vertical FOV.
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

// GetOrthoSize returns the orthographic size of the map.
func (m *Map) GetOrthoSize() float32 {
	return m.orthoSize
}

// GetProjPtr returns a pointer to the projection matrix of the map.
func (m *Map) GetProjPtr() *float32 {
	return m.projPtr
}

// GetViewPtr returns a pointer to the view matrix stored in the Map structure.
func (m *Map) GetViewPtr() *float32 {
	return m.viewPtr
}

// GetInvViewPtr returns a pointer to the inverse view matrix of the map as a float32.
func (m *Map) GetInvViewPtr() *float32 {
	return m.invViewPtr
}

// GetRoomZNear retrieves the near clipping plane value used for the room's projection matrix.
func (m *Map) GetRoomZNear() float32 {
	return m.roomZNear
}

// GetRoomZFar retrieves the far clipping plane distance for the room projection matrix.
func (m *Map) GetRoomZFar() float32 {
	return m.roomZFar
}

// GetLightCamY returns the Y-coordinate of the light camera position.
func (m *Map) GetLightCamY() float32 {
	return m.lightCamY
}

// GetMapCenterX returns the X-coordinate of the map center stored in the Map structure.
func (m *Map) GetMapCenterX() float32 {
	return m.mapCenterX
}

// GetMapCenterZ returns the Z-coordinate of the map's center point.
func (m *Map) GetMapCenterZ() float32 {
	return m.mapCenterZ
}

// GetRoomSpacePtr returns a pointer to the room space matrix.
func (m *Map) GetRoomSpacePtr() *float32 {
	return m.roomSpacePtr
}

// GetMainViewPtr returns a pointer to the main view matrix of the map as a float32.
func (m *Map) GetMainViewPtr() *float32 {
	return m.mainViewPtr
}

// SetOrthoSize sets the orthographic size, near plane, and far plane values and updates the room projection matrix.
func (m *Map) SetOrthoSize(orthoSize, zNearRoom, zFarRoom float32) {
	m.orthoSize = orthoSize
	m.roomZNear = zNearRoom
	m.roomZFar = zFarRoom
	m.updateRoomProj(m.orthoSize, m.roomZNear, m.roomZFar)
}

// SetMapCenter updates the map's center coordinates (cx, cz) and light camera height (lightCamY), and refreshes the room view.
func (m *Map) SetMapCenter(cx float32, cz float32, lightCamY float32) {
	m.mapCenterX = cx
	m.lightCamY = lightCamY
	m.mapCenterZ = cz
	m.updateRoomView(m.mapCenterX, m.lightCamY, m.mapCenterZ)
}

func (m *Map) Update(vi *model.ViewMatrix, scaleX float32, scaleY float32) {
	m.createRoomSpace(vi)
	m.createProjView(vi, scaleX, scaleY)
}

// CreateRoomSpace configures the main view matrix using the provided ViewMatrix instance to establish room spatial mapping.
func (m *Map) createRoomSpace(vi *model.ViewMatrix) {
	// Clean extraction (World Space: Z-UP)
	// Setup Camera (Main View)
	sinA, cosA := vi.GetAngleFull()
	fX, fY, fZ := float32(cosA), float32(0.0), float32(-sinA)
	rX, rY, rZ := -fZ, float32(0.0), fX
	uX, uY, uZ := float32(0.0), float32(1.0), float32(0.0)
	wX, wY, wZ := vi.GetView()
	// Spatial mapping for OpenGL (X, Z, -Y)
	camX, camY, camZ := float32(wX), float32(wZ), float32(-wY)
	m.mainView[0] = rX
	m.mainView[1] = uX
	m.mainView[2] = -fX
	m.mainView[3] = 0
	m.mainView[4] = rY
	m.mainView[5] = uY
	m.mainView[6] = -fY
	m.mainView[7] = 0
	m.mainView[8] = rZ
	m.mainView[9] = uZ
	m.mainView[10] = -fZ
	m.mainView[11] = 0
	m.mainView[12] = -dotProduct(rX, rY, rZ, camX, camY, camZ)
	m.mainView[13] = -dotProduct(uX, uY, uZ, camX, camY, camZ)
	m.mainView[14] = dotProduct(fX, fY, fZ, camX, camY, camZ)
	m.mainView[15] = 1
}

// updateRoomProj updates the room projection matrix based on orthographic size, near, and far room depth values.
func (m *Map) updateRoomProj(orthoSize, zNearRoom, zFarRoom float32) {
	if orthoSize == 0 {
		orthoSize = 1.0
	}
	diffZ := zFarRoom - zNearRoom
	if diffZ == 0 {
		diffZ = 1.0
	}
	m.roomProj[0] = 1.0 / orthoSize
	m.roomProj[1] = 0
	m.roomProj[2] = 0
	m.roomProj[3] = 0
	m.roomProj[4] = 0
	m.roomProj[5] = 1.0 / orthoSize
	m.roomProj[6] = 0
	m.roomProj[7] = 0
	m.roomProj[8] = 0
	m.roomProj[9] = 0
	m.roomProj[10] = -ndcRange / diffZ
	m.roomProj[11] = 0
	m.roomProj[12] = 0
	m.roomProj[13] = 0
	m.roomProj[14] = -(zFarRoom + zNearRoom) / diffZ
	m.roomProj[15] = 1
	MatrixMultiply4x4Ptr(m.roomSpacePtr, m.roomProjPtr, m.roomViewPtr)
}

// updateRoomView updates the transformation matrix for the room view based on the specified light camera position.
func (m *Map) updateRoomView(lX, lY, lZ float32) {
	const skew = 0.02
	m.roomView[0] = 1
	m.roomView[1] = 0
	m.roomView[2] = 0
	m.roomView[3] = 0
	m.roomView[4] = skew
	m.roomView[5] = skew
	m.roomView[6] = 1
	m.roomView[7] = 0
	m.roomView[8] = 0
	m.roomView[9] = -1
	m.roomView[10] = 0
	m.roomView[11] = 0
	m.roomView[12] = -lX
	m.roomView[13] = lY
	m.roomView[14] = -lZ
	m.roomView[15] = 1
	MatrixMultiply4x4Ptr(m.roomSpacePtr, m.roomProjPtr, m.roomViewPtr)
}

// CreateProjView computes and updates projection, view, and inverse view matrices for 3D rendering.
// It uses the provided view matrix and scaling factors for calculations and returns the matrices.
func (m *Map) createProjView(vi *model.ViewMatrix, scaleX float32, scaleY float32) {
	// Acquire angles
	sinY, cosY := vi.GetAngleFull()
	pitch := -vi.GetPitch()
	roll := vi.GetRoll()
	// Sine and Cosine of Pitch and Roll
	sinP, cosP := math.Sin(pitch), math.Cos(pitch)
	sinR, cosR := math.Sin(roll), math.Cos(roll)
	// Calculate camera base vectors (Quake-style True 3D)
	// Start from pure Yaw orientation mapped for OpenGL, and apply Pitch (up/down)
	// Forward vector (The direction the camera is looking)
	fX := float32(cosY * cosP)
	fY := float32(sinP)
	fZ := float32(-sinY * cosP)
	// Temporary Up vector (Tilted by Pitch, but without Roll)
	upX := float32(-cosY * sinP)
	upY := float32(cosP)
	upZ := float32(sinY * sinP)
	// Temporary Right vector (Always parallel to the floor before Roll)
	rightX := float32(sinY)
	rightY := float32(0)
	rightZ := float32(cosY)
	// Apply Roll (Bobbing/Tilt), rotate Right and Up vectors around the Forward axis
	rX := rightX*float32(cosR) + upX*float32(sinR)
	rY := rightY*float32(cosR) + upY*float32(sinR)
	rZ := rightZ*float32(cosR) + upZ*float32(sinR)
	uX := upX*float32(cosR) - rightX*float32(sinR)
	uY := upY*float32(cosR) - rightY*float32(sinR)
	uZ := upZ*float32(cosR) - rightZ*float32(sinR)
	// Spatial Mapping and Translation
	// Transform position from Model (Z-Up) to OpenGL space (Y-Up)
	viX, viY, viZ := vi.GetView()
	ex := float32(viX)
	ey := float32(viZ)
	ez := float32(-viY)
	// View Matrix Translation (Inverse dot product)
	tx := -(rX*ex + rY*ey + rZ*ez)
	ty := -(uX*ex + uY*ey + uZ*ez)
	tz := fX*ex + fY*ey + fZ*ez // Note: positive because OpenGL looks toward -F
	// Pure Projection Matrix (No Pitch Shearing)
	zFarRoom := m.roomZFar
	zNearRoom := m.roomZNear
	m.proj[0], m.proj[1], m.proj[2], m.proj[3] = scaleX, 0, 0, 0
	m.proj[4], m.proj[5], m.proj[6], m.proj[7] = 0, scaleY, 0, 0
	m.proj[8], m.proj[9], m.proj[10], m.proj[11] = 0, 0, (zFarRoom+zNearRoom)/(zNearRoom-zFarRoom), -1
	m.proj[12], m.proj[13], m.proj[14], m.proj[15] = 0, 0, (2*zFarRoom*zNearRoom)/(zNearRoom-zFarRoom), 0
	// View Matrix (Standard OpenGL Column-Major Layout)
	m.view[0], m.view[1], m.view[2], m.view[3] = rX, uX, -fX, 0
	m.view[4], m.view[5], m.view[6], m.view[7] = rY, uY, -fY, 0
	m.view[8], m.view[9], m.view[10], m.view[11] = rZ, uZ, -fZ, 0
	m.view[12], m.view[13], m.view[14], m.view[15] = tx, ty, tz, 1
	// Matrix inversion (Useful for dynamic Skyboxes or advanced Frustum Culling)
	if inv, ok := MatrixInverse4x4(m.view); ok {
		copy(m.invView[:], inv[:])
	}
}

/*
// UpdateUniforms2d updates the 2D projection and view matrices based on the provided view matrix and scaling factors.
// Returns the updated projection matrix, view matrix, and the inverse view matrix.
func (m *Map) UpdateUniforms2d(vi *model.ViewMatrix, scaleX float32, scaleY float32) ([16]float32, [16]float32, [16]float32) {
	pitchShear := float32(-vi.GetPitch())
	sinA, cosA := vi.GetAngleFull()
	// Base Forward (Z) and Right (X) vectors from Yaw only
	fX, fZ := float32(cosA), float32(-sinA)
	rX, rZ := float32(sinA), float32(cosA)
	// Roll
	roll := float32(vi.GetRoll())
	sinR, cosR := float32(math.Sin(float64(roll))), float32(math.Cos(float64(roll)))
	// Rotate Right (X) and Up (Y) vectors around the Forward (Z) axis
	// Original local Up was (0, 1, 0)
	// Original local Right was (rX, 0, rZ)
	// New Right vector (X)
	newRx := rX * cosR
	newRy := sinR
	newRz := rZ * cosR
	// New Up vector (Y)
	newUx := -rX * sinR
	newUy := cosR
	newUz := -rZ * sinR

	viX, viY, viZ := vi.GetView()
	ex, ey, ez := float32(viX), float32(viZ), float32(-viY)
	// Translation uses the new oriented vectors to shift the world
	tx := -(newRx*ex + newRy*ey + newRz*ez)
	ty := -(newUx*ex + newUy*ey + newUz*ez)
	// Up/Right rotate around Forward, Dir is unchanged
	tz := -((-fX)*ex + (-fZ)*ez)
	zFarRoom := m.GetRoomZFar()
	zNearRoom := m.GetRoomZNear()
	m.proj = [16]float32{
		-scaleX, 0, 0, 0,
		0, scaleY, 0, 0,
		0, pitchShear, (zFarRoom + zNearRoom) / (zNearRoom - zFarRoom), -1,
		0, 0, (2 * zFarRoom * zNearRoom) / (zNearRoom - zFarRoom), 0,
	}
	// Updated View Matrix (Column-Major)
	m.view = [16]float32{
		newRx, newUx, -fX, 0, // Col 0
		newRy, newUy, 0, 0, // Col 1
		newRz, newUz, -fZ, 0, // Col 2
		tx, ty, tz, 1, // Col 3
	}
	var invView [16]float32
	if inv, ok := MatrixInverse4x4(m.view); ok {
		invView = inv
	}
	return m.proj, m.view, invView
}
*/

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
