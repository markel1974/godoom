package shaders

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model"
)

// MapMetrics represents configuration parameters for rendering spaces, including room and flashlight projection metrics.
type MapMetrics struct {
	OrthoSize      float32
	ShadowMapRes   float32
	ZNearRoom      float32
	ZFarRoom       float32
	ZNearFlash     float32
	ZFarFlash      float32
	LightCamY      float32
	MapCenterX     float32
	MapCenterZ     float32
	FovFlashDeg    float32
	FovScaleFactor float32
	RayUnit        float32
}

// NewMapMetrics initializes and returns a pointer to a new MapMetrics instance with default metric values.
func NewMapMetrics() *MapMetrics {
	return &MapMetrics{
		OrthoSize:      640.0,
		ShadowMapRes:   1024.0,
		ZNearRoom:      1.0,
		ZFarRoom:       8192.0,
		ZNearFlash:     0.1,
		ZFarFlash:      2048.0,
		FovFlashDeg:    85.0,
		FovScaleFactor: 2.0,
		RayUnit:        512.0,
	}
}

// CreateSpaces calculates two transformation spaces for room and flashlight perspective using input view matrix and offsets.
func (m *MapMetrics) CreateSpaces(vi *model.ViewMatrix, pX, pY float64, flashOffsetX, flashOffsetY float32) ([16]float32, [16]float32) {
	//texelSize := float64((m.OrthoSize * 2.0) / m.ShadowMapRes)
	//snappedX := math.Floor(pX/texelSize) * texelSize
	//snappedY := math.Floor(-pY/texelSize) * texelSize

	// 1. Matrice Stanza (Dinamica)
	roomProj := [16]float32{
		1.0 / m.OrthoSize, 0, 0, 0,
		0, 1.0 / m.OrthoSize, 0, 0,
		0, 0, -2.0 / (m.ZFarRoom - m.ZNearRoom), 0,
		0, 0, -(m.ZFarRoom + m.ZNearRoom) / (m.ZFarRoom - m.ZNearRoom), 1,
	}

	const skew = 0.02
	//lX, lY, lZ := float32(snappedX), m.ZFarRoom/2.0, float32(snappedY)

	lX, lY, lZ := m.MapCenterX, m.LightCamY, m.MapCenterZ

	roomView := [16]float32{
		1, 0, 0, 0,
		skew, skew, 1, 0,
		0, -1, 0, 0,
		-lX, lY, -lZ, 1,
	}
	roomSpace := MatrixMultiply4x4(roomProj, roomView)

	// 2. Setup Camera
	fovScaleY := m.FovScaleFactor * float32(model.VFov)
	sinA, cosA := vi.GetAngle()
	camX, camY := vi.GetXY()
	camZ := vi.GetZ()
	pitchShear := float32(-vi.GetYaw())
	flashDirY := pitchShear / fovScaleY

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

	// 3. Spazio Torcia Locale
	posViewX, posViewY, posViewZ := flashOffsetX, flashOffsetY, float32(0.0)
	targetViewX, targetViewY, targetViewZ := float32(0.0), flashDirY*m.RayUnit, -m.RayUnit

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

	flashViewLocal := [16]float32{
		rrX, uuX, -ffX, 0, rrY, uuY, -ffY, 0, rrZ, uuZ, -ffZ, 0, tLocX, tLocY, tLocZ, 1,
	}

	flashView := MatrixMultiply4x4(flashViewLocal, mainView)

	// 4. Matrice Torcia (Dinamica)
	fovFlashRad := float32((m.FovFlashDeg * math.Pi) / 180.0)
	f := float32(1.0 / math.Tan(float64(fovFlashRad/2.0)))
	flashProj := [16]float32{
		f, 0, 0, 0,
		0, f, 0, 0,
		0, 0, (m.ZFarFlash + m.ZNearFlash) / (m.ZNearFlash - m.ZFarFlash), -1,
		0, 0, (2 * m.ZFarFlash * m.ZNearFlash) / (m.ZNearFlash - m.ZFarFlash), 0,
	}

	flashSpace := MatrixMultiply4x4(flashProj, flashView)

	return roomSpace, flashSpace
}

// MatrixMultiply4x4 multiplies two 4x4 matrices `a` and `b` and returns the resulting 4x4 matrix.
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

// MatrixInverse4x4 computes the inverse of a 4x4 matrix represented as a flat array of 16 float32 elements.
// Returns the inverted matrix and a boolean indicating success (true) or failure (false) if the matrix is non-invertible.
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
