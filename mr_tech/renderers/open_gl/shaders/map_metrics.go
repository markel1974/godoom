package shaders

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model"
)

// MapMetrics is a structure that defines various metrics and configurations for room projection and flashlight rendering.
type MapMetrics struct {
	orthoSize      float32
	roomZNear      float32
	roomZFar       float32
	lightCamY      float32
	mapCenterX     float32
	mapCenterZ     float32
	flashZNear     float32
	flashZFar      float32
	flashFovDeg    float32
	flashFov       float32
	flashConeStart float32
	flashConeEnd   float32
	flashAspect    float32
	fovScaleFactor float32
	scaleFovY      float32

	roomProj  [16]float32
	roomView  [16]float32
	roomSpace [16]float32
	flashProj [16]float32
}

// NewMapMetrics initializes and returns a new instance of MapMetrics with predefined settings.
func NewMapMetrics() *MapMetrics {
	m := &MapMetrics{}
	m.SetOrthoSize(640.0, 0.0, 8192.0)
	m.SetFlash(85.0, 0.1, 2048.0, 1024, 1024)
	m.SetMapCenter(0.0, 0.0, 0.0)
	m.SetFovScaleFactor(2.0)
	return m
}

// GetOrthoSize returns the orthographic size value from the MapMetrics instance.
func (m *MapMetrics) GetOrthoSize() float32 {
	return m.orthoSize
}

// SetOrthoSize sets the orthogonal size and the near and far clipping planes for the room projection matrix.
func (m *MapMetrics) SetOrthoSize(orthoSize, zNearRoom, zFarRoom float32) {
	m.orthoSize = orthoSize
	m.roomZNear = zNearRoom
	m.roomZFar = zFarRoom
	m.updateRoomProj(m.orthoSize, m.roomZNear, m.roomZFar)
}

// GetRoomZNear retrieves the near clipping plane distance for the room's projection matrix.
func (m *MapMetrics) GetRoomZNear() float32 {
	return m.roomZNear
}

// GetRoomZFar retrieves the far clipping distance for the room in the projection matrix.
func (m *MapMetrics) GetRoomZFar() float32 {
	return m.roomZFar
}

// GetFlashZNear returns the near clipping distance for the flashlight projection.
func (m *MapMetrics) GetFlashZNear() float32 {
	return m.flashZNear
}

// GetFlashAspect retrieves the aspect ratio used for the flash projection.
func (m *MapMetrics) GetFlashAspect() float32 {
	return m.flashAspect
}

// SetFlash configures the flashlight's field of view, near and far plane distances, dimensions, and projection matrix.
func (m *MapMetrics) SetFlash(flashFovDeg, zNearFlash, zFarFlash, width, height float32) {
	m.flashFovDeg = flashFovDeg
	m.flashZNear = zNearFlash
	m.flashZFar = zFarFlash
	m.flashAspect = width / height
	fovFlashRad := (m.flashFovDeg * math.Pi) / 180.0
	m.flashFov = float32(1.0 / math.Tan(float64(fovFlashRad/2.0)))
	m.updateFlashProj(m.flashFov, m.flashAspect, m.flashZNear, m.flashZFar)
	m.flashConeStart = float32(math.Cos(float64(m.flashFovDeg)/2.0*math.Pi/180.0)) + 0.01
	m.flashConeEnd = float32(math.Cos(float64(m.flashFovDeg) / 2.0 * 0.6 * math.Pi / 180.0))
}

// GetFlashZFar returns the far clipping distance of the flashlight projection matrix.
func (m *MapMetrics) GetFlashZFar() float32 {
	return m.flashZFar
}

// SetMapCenter sets the map center coordinates and light camera Y position, updating the view matrix accordingly.
func (m *MapMetrics) SetMapCenter(cx float32, cz float32, lightCamY float32) {
	m.mapCenterX = cx
	m.mapCenterZ = cz
	m.lightCamY = lightCamY
	m.updateRoomView(m.mapCenterX, m.lightCamY, m.mapCenterZ)
}

// GetLightCamY returns the current Y-coordinate of the light camera in the map metrics.
func (m *MapMetrics) GetLightCamY() float32 {
	return m.lightCamY
}

// GetMapCenterX returns the X-coordinate of the map center stored in the MapMetrics instance.
func (m *MapMetrics) GetMapCenterX() float32 {
	return m.mapCenterX
}

// GetMapCenterZ returns the Z-coordinate of the center of the map stored in the MapMetrics instance.
func (m *MapMetrics) GetMapCenterZ() float32 {
	return m.mapCenterZ
}

// GetFovFlashDeg returns the field of view (FOV) of the flashlight in degrees.
func (m *MapMetrics) GetFovFlashDeg() float32 {
	return m.flashFovDeg
}

// GetFovScaleFactor retrieves the scaling factor applied to the field of view (FOV) for perspective calculations.
func (m *MapMetrics) GetFovScaleFactor() float32 {
	return m.fovScaleFactor
}

// GetFlashConeStart retrieves the starting angle's cosine value for the flashlight cone, used for defining its shape.
func (m *MapMetrics) GetFlashConeStart() float32 {
	return m.flashConeStart
}

// GetFlashConeEnd returns the ending value of the flashlight cone's angle, used for cone-based illumination calculations.
func (m *MapMetrics) GetFlashConeEnd() float32 {
	return m.flashConeEnd
}

// SetFovScaleFactor sets the field-of-view (FOV) scale factor and updates the scaled FOV value.
func (m *MapMetrics) SetFovScaleFactor(value float32) {
	m.fovScaleFactor = value
	m.scaleFovY = m.fovScaleFactor * float32(model.VFov)
}

// updateRoomProj updates the orthographic projection matrix (roomProj) and combined matrix (roomSpace) for the room space.
// It recalculates these matrices based on the provided orthoSize, zNearRoom, and zFarRoom values.
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
		0, 0, -2.0 / diffZ, 0,
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

// updateFlashProj updates the flash projection matrix based on field of view, aspect ratio, and near/far clipping planes.
func (m *MapMetrics) updateFlashProj(flashFov, flashAspect, zNearFlash, zFarFlash float32) {
	diffZ := zNearFlash - zFarFlash
	if diffZ == 0 {
		diffZ = 1.0
	}
	if flashAspect == 0 {
		flashAspect = 1.0
	}
	m.flashProj = [16]float32{
		flashFov / flashAspect, 0, 0, 0,
		0, flashFov, 0, 0,
		0, 0, (zFarFlash + zNearFlash) / diffZ, -1,
		0, 0, (2 * zFarFlash * zNearFlash) / diffZ, 0,
	}
}

// CreateSpaces generates two transformation matrices: room space and flashlight space, based on the given parameters.
func (m *MapMetrics) CreateSpaces(vi *model.ViewMatrix, flashOffsetX, flashOffsetY float32) ([16]float32, [16]float32) {
	// 1. Setup Camera
	sinA, cosA := vi.GetAngle()
	camX, camY := vi.GetXY()
	camZ := vi.GetZ()
	pitchShear := float32(-vi.GetYaw())
	flashDirY := pitchShear / m.scaleFovY

	fX, fY, fZ := float32(cosA), float32(0.0), float32(-sinA)
	rX, rY, rZ := -fZ, float32(0.0), fX
	uX, uY, uZ := float32(0.0), float32(1.0), float32(0.0)
	eX, eY, eZ := float32(camX), float32(camZ), float32(-camY)

	tx := -(rX*eX + rY*eY + rZ*eZ)
	ty := -(uX*eX + uY*eY + uZ*eZ)
	tz := fX*eX + fY*eY + fZ*eZ

	mainView := [16]float32{
		rX, uX, -fX, 0, rY, uY, -fY, 0, rZ, uZ, -fZ, 0, tx, ty, tz, 1,
	}

	// 2. Spazio Torcia Locale
	posViewX, posViewY, posViewZ := flashOffsetX, flashOffsetY, float32(0.0)
	targetViewX, targetViewY, targetViewZ := float32(0.0), flashDirY*m.flashZFar, -m.flashZFar

	ffX, ffY, ffZ := targetViewX-posViewX, targetViewY-posViewY, targetViewZ-posViewZ
	invLenF := float32(1.0 / math.Sqrt(float64(ffX*ffX+ffY*ffY+ffZ*ffZ)))
	ffX, ffY, ffZ = ffX*invLenF, ffY*invLenF, ffZ*invLenF

	upViewX, upViewY, upViewZ := float32(0.0), float32(1.0), float32(0.0)

	rrX := ffY*upViewZ - ffZ*upViewY
	rrY := ffZ*upViewX - ffX*upViewZ
	rrZ := ffX*upViewY - ffY*upViewX
	invLenR := float32(1.0 / math.Sqrt(float64(rrX*rrX+rrY*rrY+rrZ*rrZ)))
	rrX, rrY, rrZ = rrX*invLenR, rrY*invLenR, rrZ*invLenR

	uuX := rrY*ffZ - rrZ*ffY
	uuY := rrZ*ffX - rrX*ffZ
	uuZ := rrX*ffY - rrY*ffX

	tLocX := -(rrX*posViewX + rrY*posViewY + rrZ*posViewZ)
	tLocY := -(uuX*posViewX + uuY*posViewY + uuZ*posViewZ)
	tLocZ := ffX*posViewX + ffY*posViewY + ffZ*posViewZ

	// 3. Matrice Torcia (Dinamica)
	flashViewLocal := [16]float32{
		rrX, uuX, -ffX, 0, rrY, uuY, -ffY, 0, rrZ, uuZ, -ffZ, 0, tLocX, tLocY, tLocZ, 1,
	}
	flashView := MatrixMultiply4x4(flashViewLocal, mainView)
	flashSpace := MatrixMultiply4x4(m.flashProj, flashView)

	return m.roomSpace, flashSpace
}

// MatrixMultiply4x4 multiplies two 4x4 matrices represented as 1D arrays of 16 floats and returns the resulting matrix.
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

// MatrixInverse4x4 computes the inverse of a 4x4 matrix represented as a 16-element array.
// Returns the inverted matrix and a boolean indicating success. If the determinant is 0, the inversion fails.
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
