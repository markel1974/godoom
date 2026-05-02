package shaders

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model"
)

// The 2.0 is the extent of OpenGL's NDC range [-1.0, +1.0]
const ndcRange = 2.0

// Helper inline per evitare boilerplate matematico
func cross(ax, ay, az, bx, by, bz float32) (float32, float32, float32) {
	return ay*bz - az*by, az*bx - ax*bz, ax*by - ay*bx
}

func normalize(x, y, z float32) (float32, float32, float32) {
	invLen := float32(1.0 / math.Sqrt(float64(x*x+y*y+z*z)))
	return x * invLen, y * invLen, z * invLen
}

func dot(ax, ay, az, bx, by, bz float32) float32 {
	return ax*bx + ay*by + az*bz
}

// MapMetrics is a structure that defines various metrics and configurations for room projection and flashlight rendering.
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

// NewMapMetrics initializes and returns a new instance of MapMetrics with predefined settings.
func NewMapMetrics(flash *model.Flash) *MapMetrics {
	m := &MapMetrics{
		flash: flash,
	}
	m.SetOrthoSize(float32(640), 0.0, 8192.0)
	m.Rebuild(1024, 1024)
	m.SetMapCenter(0.0, 0.0, 0.0)
	return m
}

// GetScale2d calculates and returns the scaled horizontal and vertical 2D projection factors based on input dimensions.
func (m *MapMetrics) GetScale2d(width, height int32) (float32, float32) {
	aspect := float32(width) / float32(height)
	scaleX := (ndcRange / aspect) * float32(model.HFov)
	scaleY := ndcRange * float32(model.VFov)
	return scaleX, scaleY
}

// GetScale3d calculates and returns the X and Y scaling factors based on horizontal FOV.
func (m *MapMetrics) GetScale3d(width, height int32, aspectRatio float32, fovVerticalDegrees float32) (float32, float32) {
	// 1. Aspect ratio della finestra (es. 1280/960 = 1.333)
	aspect := float32(width) / float32(height)
	// 2. Field of View Verticale (75.0 è lo standard Quake/Retro per schermi 4:3)
	// Se su schermi 16:9 ti sembra troppo "zoomato", alzalo a 80 o 90.
	//const fovVerticalDegrees = 80.0
	fovYRad := (fovVerticalDegrees * math.Pi) / 180.0
	// 3. Focale base derivata dall'angolo visivo
	focal := float32(1.0 / math.Tan(float64(fovYRad)/2.0))
	// 4. Mappatura canonica OpenGL (Perspective Projection)
	// L'asse Y usa la focale pura. L'asse X viene corretto (diviso) dall'aspect ratio.
	scaleY := focal * aspectRatio
	scaleX := focal / aspect
	return scaleX, scaleY
}

// GetShadowSize returns the width and height of the shadow map as two int32 values.
func (m *MapMetrics) GetShadowSize() (int32, int32) {
	return m.shadowWidth, m.shadowHeight
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

// GetFlashAspect retrieves the aspect ratio used for the flash projection.
func (m *MapMetrics) GetFlashAspect() float32 {
	return m.shadowAspect
}

// Rebuild configures the flashlight's field of view, near and far plane distances, dimensions, and projection matrix.
func (m *MapMetrics) Rebuild(width, height int32) {
	m.shadowWidth = width
	m.shadowHeight = height
	m.shadowAspect = float32(width) / float32(height)
	if m.shadowAspect == 0 {
		m.shadowAspect = 1.0
	}
	m.flash.Rebuild(ndcRange)
	m.updateFlashProj()
}

// SetMapCenter sets the map center coordinates and light camera Y position, updating the view matrix accordingly.
func (m *MapMetrics) SetMapCenter(cx float32, cz float32, lightCamY float32) {
	m.mapCenterX = cx
	m.lightCamY = lightCamY
	m.mapCenterZ = cz
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

// updateFlashProj updates the flash projection matrix based on field of view, aspect ratio, and near/far clipping planes.
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

// CreateSpaces2d generates and returns the room space matrix, flashlight space matrix, and main view matrix based on input parameters.
func (m *MapMetrics) CreateSpaces2d(vi *model.ViewMatrix, flashOffsetX, flashOffsetY float32) ([16]float32, [16]float32, [16]float32) {
	// Clean extraction (World Space: Z-UP)
	// Setup Camera (Main View)
	sinA, cosA := vi.GetAngleFull()
	fX, fY, fZ := float32(cosA), float32(0.0), float32(-sinA)
	rX, rY, rZ := -fZ, float32(0.0), fX
	uX, uY, uZ := float32(0.0), float32(1.0), float32(0.0)
	wX, wY, wZ := vi.GetXYZ()
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

// CreateRoomSpace generates the room's spatial configuration and main view matrix based on the provided view matrix parameters.
func (m *MapMetrics) CreateRoomSpace(vi *model.ViewMatrix) ([16]float32, [16]float32) {
	// Clean extraction (World Space: Z-UP)
	// Setup Camera (Main View)
	sinA, cosA := vi.GetAngleFull()
	fX, fY, fZ := float32(cosA), float32(0.0), float32(-sinA)
	rX, rY, rZ := -fZ, float32(0.0), fX
	uX, uY, uZ := float32(0.0), float32(1.0), float32(0.0)
	wX, wY, wZ := vi.GetXYZ()
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

// CreateFlashSpace generates a 4x4 transformation matrix for the flashlight's view space based on input parameters.
// It calculates the local flashlight position, orientation, and combines it with the main view and projection matrices.
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

// CreateSpotLightSpace calculates the light-space transformation matrix for a spotlight.
// It combines a perspective projection matrix and a view matrix based on the position, direction, and frustum parameters.
func (m *MapMetrics) CreateSpotLightSpace(posX, posY, posZ, dirX, dirY, dirZ float32, fovDeg, near, far float32) [16]float32 {
	// 1. Matrice di Proiezione (Perspective)
	// Per una mappa delle ombre, l'aspect ratio è rigorosamente 1.0 (è quadrata)
	fovRad := fovDeg * math.Pi / 180.0
	f := float32(1.0 / math.Tan(float64(fovRad)/2.0))
	proj := [16]float32{
		f, 0, 0, 0,
		0, f, 0, 0,
		0, 0, (far + near) / (near - far), -1.0,
		0, 0, (2.0 * far * near) / (near - far), 0,
	}

	// 2. Matrice di Vista (LookAt)
	ffX, ffY, ffZ := normalize(dirX, dirY, dirZ)

	// Vettore UP standard (Y-up in OpenGL)
	upX, upY, upZ := float32(0.0), float32(1.0), float32(0.0)

	// Sicurezza Anti-Gimbal-Lock: se il faretto punta dritto in alto o in basso (pavimento/soffitto)
	// il prodotto incrociato fallirebbe. Usiamo -Z come UP alternativo.
	if math.Abs(float64(ffY)) > 0.999 {
		upX, upY, upZ = 0.0, 0.0, -1.0
	}

	// R = Right, U = Upricalcolato
	rrX, rrY, rrZ := normalize(cross(ffX, ffY, ffZ, upX, upY, upZ))
	uuX, uuY, uuZ := cross(rrX, rrY, rrZ, ffX, ffY, ffZ) // Già normalizzato

	// Traslazione negativa (dot product tra assi invertiti e posizione)
	tX := -dot(rrX, rrY, rrZ, posX, posY, posZ)
	tY := -dot(uuX, uuY, uuZ, posX, posY, posZ)
	tZ := dot(ffX, ffY, ffZ, posX, posY, posZ)

	view := [16]float32{
		rrX, uuX, -ffX, 0,
		rrY, uuY, -ffY, 0,
		rrZ, uuZ, -ffZ, 0,
		tX, tY, tZ, 1,
	}

	// 3. Spazio Finale della Luce (Proj * View)
	return MatrixMultiply4x4(proj, view)
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

// GetFovScaleFactor retrieves the scaling factor applied to the field of view (FOV) for perspective calculations.
//func (m *MapMetrics) GetFovScaleFactor() float32 {
//	return m.fovScaleFactor
//}

// SetFovScaleFactor sets the field-of-view (FOV) scale factor and updates the scaled FOV value.
//func (m *MapMetrics) SetFovScaleFactor(value float32) {
//	m.fovScaleFactor = value
//	m.scaleFovY = m.fovScaleFactor * float32(model.VFov)
//}
