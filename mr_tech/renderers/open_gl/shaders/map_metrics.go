package shaders

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model"
)

// ndcRange represents the normalized device coordinate range used for various scaling and projection calculations.
const ndcRange = 2.0

// cross calculates the cross product of two 3D vectors (a and b) and returns the resulting vector components.
func cross(ax, ay, az, bx, by, bz float32) (float32, float32, float32) {
	return ay*bz - az*by, az*bx - ax*bz, ax*by - ay*bx
}

// normalize calculates the unit vector of the input vector (x, y, z) to normalize its magnitude to 1.
func normalize(x, y, z float32) (float32, float32, float32) {
	invLen := float32(1.0 / math.Sqrt(float64(x*x+y*y+z*z)))
	return x * invLen, y * invLen, z * invLen
}

// dot calculates the dot product of two 3D vectors defined by their components.
func dot(ax, ay, az, bx, by, bz float32) float32 {
	return ax*bx + ay*by + az*bz
}

// MapMetrics represents settings and transformations for rendering maps, including orthographic size, view parameters, and shadows.
type MapMetrics struct {
	orthoSize    float32
	roomZNear    float32
	roomZFar     float32
	lightCamY    float32
	mapCenterX   float32
	mapCenterZ   float32
	shadowWidth  int32
	shadowHeight int32
	shadowAspect float32
	roomProj     [16]float32
	roomView     [16]float32
	roomSpace    [16]float32
	flashProj    [16]float32
	flash        *model.Flash
}

// NewMapMetrics initializes and returns a pointer to a MapMetrics instance configured using the given Flash object.
func NewMapMetrics(flash *model.Flash) *MapMetrics {
	m := &MapMetrics{
		flash: flash,
	}
	m.SetOrthoSize(float32(640), 0.0, 8192.0)
	m.Rebuild(1024, 1024)
	m.SetMapCenter(0.0, 0.0, 0.0)
	return m
}

// GetScale2d calculates the horizontal and vertical scales based on the given width and height of the viewport.
func (m *MapMetrics) GetScale2d(width, height int32) (float32, float32) {
	aspect := float32(width) / float32(height)
	scaleX := (ndcRange / aspect) * float32(model.HFov)
	scaleY := ndcRange * float32(model.VFov)
	return scaleX, scaleY
}

// GetScale3d calculates and returns the scaling factors for 3D projection based on screen dimensions, aspect ratio, and FOV.
func (m *MapMetrics) GetScale3d(width, height int32, aspectRatio float32, fovVerticalDegrees float32) (float32, float32) {
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

// GetShadowSize returns the shadow map's width and height as two int32 values.
func (m *MapMetrics) GetShadowSize() (int32, int32) {
	return m.shadowWidth, m.shadowHeight
}

// GetOrthoSize retrieves the orthographic size associated with the current MapMetrics instance.
func (m *MapMetrics) GetOrthoSize() float32 {
	return m.orthoSize
}

// SetOrthoSize sets the orthographic size and near/far room clipping planes, updating the room projection matrix.
func (m *MapMetrics) SetOrthoSize(orthoSize, zNearRoom, zFarRoom float32) {
	m.orthoSize = orthoSize
	m.roomZNear = zNearRoom
	m.roomZFar = zFarRoom
	m.updateRoomProj(m.orthoSize, m.roomZNear, m.roomZFar)
}

// GetRoomZNear returns the near clipping plane distance for the room projection.
func (m *MapMetrics) GetRoomZNear() float32 {
	return m.roomZNear
}

// GetRoomZFar retrieves the far clipping distance of the room's projection matrix.
func (m *MapMetrics) GetRoomZFar() float32 {
	return m.roomZFar
}

// GetFlashAspect returns the aspect ratio used for the flashlight's projection, derived from the shadow's dimensions.
func (m *MapMetrics) GetFlashAspect() float32 {
	return m.shadowAspect
}

// Rebuild updates the shadow dimensions, recalculates the shadow aspect ratio, and rebuilds related projection matrices.
func (m *MapMetrics) Rebuild(width, height int32) {
	m.shadowWidth = width   // * 0.5)
	m.shadowHeight = height // * 0.5)
	m.shadowAspect = float32(m.shadowWidth) / float32(m.shadowHeight)
	if m.shadowAspect == 0 {
		m.shadowAspect = 1.0
	}
	m.flash.Rebuild(ndcRange)
	m.updateFlashProj()
}

// SetMapCenter updates the map's center coordinates and light camera Y position, recalculating the room view matrix.
func (m *MapMetrics) SetMapCenter(cx float32, cz float32, lightCamY float32) {
	m.mapCenterX = cx
	m.lightCamY = lightCamY
	m.mapCenterZ = cz
	m.updateRoomView(m.mapCenterX, m.lightCamY, m.mapCenterZ)
}

// GetLightCamY retrieves the Y-coordinate of the light camera position stored in the MapMetrics instance.
func (m *MapMetrics) GetLightCamY() float32 {
	return m.lightCamY
}

// GetMapCenterX returns the X-coordinate of the map's center.
func (m *MapMetrics) GetMapCenterX() float32 {
	return m.mapCenterX
}

// GetMapCenterZ retrieves the Z-coordinate of the map's center position.
func (m *MapMetrics) GetMapCenterZ() float32 {
	return m.mapCenterZ
}

// updateRoomProj updates the room projection matrix and computes the room space matrix based on the given parameters.
func (m *MapMetrics) updateRoomProj(orthoSize, zNearRoom, zFarRoom float32) {
	if orthoSize == 0 {
		orthoSize = 1.0
	}
	diffZ := zFarRoom - zNearRoom
	if diffZ == 0 {
		diffZ = 1.0
	}
	m.roomProj = [16]float32{
		1.0 / orthoSize, 0, 0, 0,
		0, 1.0 / orthoSize, 0, 0,
		0, 0, -ndcRange / diffZ, 0,
		0, 0, -(zFarRoom + zNearRoom) / diffZ, 1,
	}
	m.roomSpace = MatrixMultiply4x4(m.roomProj, m.roomView)
}

// updateRoomView updates the room view matrix and recalculates the combined room space matrix.
func (m *MapMetrics) updateRoomView(lX, lY, lZ float32) {
	const skew = 0.02
	m.roomView = [16]float32{
		1, 0, 0, 0,
		skew, skew, 1, 0,
		0, -1, 0, 0,
		-lX, lY, -lZ, 1,
	}
	m.roomSpace = MatrixMultiply4x4(m.roomProj, m.roomView)
}

// updateFlashProj computes and updates the flashlight projection matrix using its field of view and near/far plane distances.
func (m *MapMetrics) updateFlashProj() {
	flashFov, zNearFlash, zFarFlash := float32(m.flash.GetFov()), float32(m.flash.GetZNear()), float32(m.flash.GetZFar())
	diffZ := zNearFlash - zFarFlash
	if diffZ == 0 {
		diffZ = 1.0
	}
	m.flashProj = [16]float32{
		flashFov / m.shadowAspect, 0, 0, 0,
		0, flashFov, 0, 0,
		0, 0, (zFarFlash + zNearFlash) / diffZ, -1,
		0, 0, (2 * zFarFlash * zNearFlash) / diffZ, 0,
	}
}

// CreateRoomSpace generates and returns the room space and main view transformation matrices based on the provided view matrix.
func (m *MapMetrics) CreateRoomSpace(vi *model.ViewMatrix) ([16]float32, [16]float32) {
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
	return m.roomSpace, mainView
}

// CreateFlashSpace computes the flashlight's transformation matrix in view space with given offset and transformation details.
func (m *MapMetrics) CreateFlashSpace(mainView [16]float32, flashOffsetX, flashOffsetY float32) [16]float32 {
	// Local ShadowLight Space (LookAt calculation)
	posViewX, posViewY, posViewZ := flashOffsetX, flashOffsetY, float32(0.0)
	targetX, targetY, targetZ := float32(0.0), float32(0.0), -float32(m.flash.GetZFar())
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
	return flashSpace
}

// CreateSpotLightSpace generates a 4x4 transformation matrix for a spotlight's view and projection in shadow mapping.
func (m *MapMetrics) CreateSpotLightSpace(posX, posY, posZ, dirX, dirY, dirZ float32, fovDeg, near, far float32) [16]float32 {
	// Projection Matrix (Perspective)
	// For a shadow map, aspect ratio is strictly 1.0 (it's square)
	fovRad := fovDeg * math.Pi / 180.0
	f := float32(1.0 / math.Tan(float64(fovRad)/2.0))
	proj := [16]float32{
		f, 0, 0, 0,
		0, f, 0, 0,
		0, 0, (far + near) / (near - far), -1.0,
		0, 0, (2.0 * far * near) / (near - far), 0,
	}
	// View Matrix (LookAt)
	ffX, ffY, ffZ := normalize(dirX, dirY, dirZ)
	// Standard UP vector (Y-up in OpenGL)
	upX, upY, upZ := float32(0.0), float32(1.0), float32(0.0)
	// Anti-Gimbal-Lock safety: if the spotlight points straight up or down (floor/ceiling)
	// the cross product would fail. Use -Z as alternative UP.
	if math.Abs(float64(ffY)) > 0.999 {
		upX, upY, upZ = 0.0, 0.0, -1.0
	}
	// R = Right, U = Recalculated Up
	rrX, rrY, rrZ := normalize(cross(ffX, ffY, ffZ, upX, upY, upZ))
	uuX, uuY, uuZ := cross(rrX, rrY, rrZ, ffX, ffY, ffZ) // Already normalized
	// Negative translation (dot product between inverted axes and position)
	tX := -dot(rrX, rrY, rrZ, posX, posY, posZ)
	tY := -dot(uuX, uuY, uuZ, posX, posY, posZ)
	tZ := dot(ffX, ffY, ffZ, posX, posY, posZ)
	view := [16]float32{
		rrX, uuX, -ffX, 0,
		rrY, uuY, -ffY, 0,
		rrZ, uuZ, -ffZ, 0,
		tX, tY, tZ, 1,
	}
	// Final Light Space (Proj * View)
	return MatrixMultiply4x4(proj, view)
}

// MatrixMultiply4x4 multiplies two 4x4 matrices represented as flat arrays and returns the resulting matrix.
func MatrixMultiply4x4(a [16]float32, b [16]float32) [16]float32 {
	var out [16]float32
	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
			sum := float32(0.0)
			for i := 0; i < 4; i++ {
				sum += a[i*4+row] * b[col*4+i]
			}
			out[col*4+row] = sum
		}
	}
	return out
}

// MatrixInverse4x4 computes the inverse of a 4x4 matrix represented as a 16-element float32 array in column-major order.
// Returns the inverted matrix and a boolean indicating success (true) or failure (false) if the determinant is zero.
func MatrixInverse4x4(m [16]float32) ([16]float32, bool) {
	var inv [16]float32
	var det float32

	inv[0] = m[5]*m[10]*m[15] - m[5]*m[11]*m[14] - m[9]*m[6]*m[15] + m[9]*m[7]*m[14] + m[13]*m[6]*m[11] - m[13]*m[7]*m[10]
	inv[4] = -m[4]*m[10]*m[15] + m[4]*m[11]*m[14] + m[8]*m[6]*m[15] - m[8]*m[7]*m[14] - m[12]*m[6]*m[11] + m[12]*m[7]*m[10]
	inv[8] = m[4]*m[9]*m[15] - m[4]*m[11]*m[13] - m[8]*m[5]*m[15] + m[8]*m[7]*m[13] + m[12]*m[5]*m[11] - m[12]*m[7]*m[9]
	inv[12] = -m[4]*m[9]*m[14] + m[4]*m[10]*m[13] + m[8]*m[5]*m[14] - m[8]*m[6]*m[13] - m[12]*m[5]*m[10] + m[12]*m[6]*m[9]
	inv[1] = -m[1]*m[10]*m[15] + m[1]*m[11]*m[14] + m[9]*m[2]*m[15] - m[9]*m[3]*m[14] - m[13]*m[2]*m[11] + m[13]*m[3]*m[10]
	inv[5] = m[0]*m[10]*m[15] - m[0]*m[11]*m[14] - m[8]*m[2]*m[15] + m[8]*m[3]*m[14] + m[12]*m[2]*m[11] - m[12]*m[3]*m[10]
	inv[9] = -m[0]*m[9]*m[15] + m[0]*m[11]*m[13] + m[8]*m[1]*m[15] - m[8]*m[3]*m[13] - m[12]*m[1]*m[11] + m[12]*m[3]*m[9]
	inv[13] = m[0]*m[9]*m[14] - m[0]*m[10]*m[13] - m[8]*m[1]*m[14] + m[8]*m[2]*m[13] + m[12]*m[1]*m[10] - m[12]*m[2]*m[9]
	inv[2] = m[1]*m[5]*m[15] - m[1]*m[7]*m[14] - m[5]*m[2]*m[15] + m[5]*m[3]*m[14] + m[13]*m[2]*m[7] - m[13]*m[3]*m[5]
	inv[6] = -m[0]*m[5]*m[15] + m[0]*m[7]*m[14] + m[4]*m[2]*m[15] - m[4]*m[3]*m[14] - m[12]*m[2]*m[7] + m[12]*m[3]*m[5]
	inv[10] = m[0]*m[5]*m[15] - m[0]*m[7]*m[13] - m[4]*m[1]*m[15] + m[4]*m[3]*m[13] + m[12]*m[1]*m[7] - m[12]*m[3]*m[5]
	inv[14] = -m[0]*m[5]*m[14] + m[0]*m[6]*m[13] + m[4]*m[1]*m[14] - m[4]*m[2]*m[13] - m[12]*m[1]*m[6] + m[12]*m[2]*m[5]
	inv[3] = -m[1]*m[6]*m[11] + m[1]*m[7]*m[10] + m[5]*m[2]*m[11] - m[5]*m[3]*m[10] - m[9]*m[2]*m[7] + m[9]*m[3]*m[6]
	inv[7] = m[0]*m[6]*m[11] - m[0]*m[7]*m[10] - m[4]*m[2]*m[11] + m[4]*m[3]*m[10] + m[8]*m[2]*m[7] - m[8]*m[3]*m[6]
	inv[11] = -m[0]*m[5]*m[11] + m[0]*m[7]*m[9] + m[4]*m[1]*m[11] - m[4]*m[3]*m[9] - m[8]*m[1]*m[7] + m[8]*m[3]*m[5]
	inv[15] = m[0]*m[5]*m[10] - m[0]*m[6]*m[9] - m[4]*m[1]*m[10] + m[4]*m[2]*m[9] + m[8]*m[1]*m[6] - m[8]*m[2]*m[5]

	det = m[0]*inv[0] + m[1]*inv[4] + m[2]*inv[8] + m[3]*inv[12]
	if det == 0 {
		return [16]float32{}, false
	}
	invDet := 1.0 / det
	for i := 0; i < 16; i++ {
		inv[i] *= invDet
	}
	return inv, true
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
