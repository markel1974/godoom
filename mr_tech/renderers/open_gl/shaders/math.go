package shaders

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model"
)

// zNearRoom defines the near clipping plane distance for room projection.
// zFarRoom defines the far clipping plane distance for room projection.
// zNearFlash defines the near clipping plane distance for flashlight projection.
// zFarFlash defines the far clipping plane distance for flashlight projection.
// fovScaleY calculates the scaled vertical field of view for room projection.
// fovFlashDeg specifies the flashlight's field of view in degrees.
// fovFlashRad converts the flashlight's field of view from degrees to radians.
// fovScaleFactor determines the scaling factor applied to the field of view.
// fovFlashHalfRad calculates half of the flashlight's field of view in radians.
// orthoSize sets the size of the orthographic directional projection for a room.
// texelSize calculates the size of a texel based on the orthographic projection size.
// flashBaseMax defines the maximum base intensity for the flashlight.
// rayUnit specifies the unit distance for a ray in the scene.
const (
	//far                           = float32(4096.0)
	zNearRoom               = 1.0
	zFarRoom                = 8192.0
	zNearFlash              = 0.1
	zFarFlash               = 2048.0
	fovScaleY               = fovScaleFactor * float32(model.VFov)
	fovFlashDeg     float64 = 130.0
	fovFlashRad             = float32((fovFlashDeg * math.Pi) / 180.0)
	fovScaleFactor          = float32(2.0)
	fovFlashHalfRad         = (fovFlashDeg / 2.0) * (math.Pi / 180.0)
	orthoSize               = 640.0 // PROIEZIONE STANZA (Ortografica Direzionale) ---
	texelSize               = (orthoSize * 2.0) / 1024.0
	flashBaseMax            = float32(0.90)
	rayUnit                 = float32(512.0)
)

// _flashConeStartMax defines the maximum starting angle for the flash cone, slightly larger than the cosine of half the FOV.
var _flashConeStartMax = float32(math.Cos(fovFlashHalfRad)) + 0.01

// _flashConeEndMax defines the maximum cosine value for the end of the flashlight cone, scaled by 0.6 of the flash FOV.
var _flashConeEndMax = float32(math.Cos(fovFlashHalfRad * 0.6))

// _f represents the scaling factor for the flash projection matrix derived from the field of view in radians.
var _f = float32(1.0 / math.Tan(float64(fovFlashRad/2.0)))

// _flashProj is a 4x4 projection matrix for flashlight rendering, configured with perspective parameters and depth range.
var _flashProj = [16]float32{
	_f, 0, 0, 0,
	0, _f, 0, 0,
	0, 0, (zFarFlash + zNearFlash) / (zNearFlash - zFarFlash), -1,
	0, 0, (2 * zFarFlash * zNearFlash) / (zNearFlash - zFarFlash), 0,
}

// _roomProj defines a 4x4 orthographic projection matrix for rendering a room in normalized device coordinates.
var _roomProj = [16]float32{
	1.0 / orthoSize, 0, 0, 0,
	0, 1.0 / orthoSize, 0, 0,
	0, 0, -2.0 / (zFarRoom - zNearRoom), 0,
	0, 0, -(zFarRoom + zNearRoom) / (zFarRoom - zNearRoom), 1,
}

// CreateSpaces computes and returns two 4x4 transformation matrices: roomSpace and flashSpace, based on input parameters and projections.
func CreateSpaces(vi *model.ViewMatrix, pX, pY float64, flashOffsetX, flashOffsetY float32) ([16]float32, [16]float32) {
	snappedX := math.Floor(pX/texelSize) * texelSize
	snappedY := math.Floor(-pY/texelSize) * texelSize

	// lY a 4096.0 impedisce il near-plane clipping dei soffitti
	// Telecamera altissima per non tagliare i soffitti
	lX, lY, lZ := float32(snappedX), float32(4096.0), float32(snappedY)
	roomView := [16]float32{
		1, 0, 0, 0,
		0.02, 0.02, 1, 0, // TILT: Inclinazione impercettibile per dare area d'ombra ai muri
		0, -1, 0, 0,
		-lX, lZ, -lY, 1,
	}

	roomSpace := MatrixMultiply4x4(_roomProj, roomView)

	// --- 2. SINCRONIZZAZIONE FOV E PARALLASSE TORCIA (Prospettica) ---
	sinA, cosA := vi.GetAngle()
	camX, camY := vi.GetXY()
	camZ := vi.GetZ()
	pitchShear := float32(-vi.GetYaw())
	flashDirY := pitchShear / fovScaleY

	// Estrazione vettori base della Telecamera
	fCamX, fCamY, fCamZ := float32(cosA), flashDirY, float32(-sinA)
	invLenF := float32(1.0 / math.Sqrt(float64(fCamX*fCamX+fCamY*fCamY+fCamZ*fCamZ)))
	fCamX, fCamY, fCamZ = fCamX*invLenF, fCamY*invLenF, fCamZ*invLenF

	rCamX, rCamY, rCamZ := -fCamZ, float32(0.0), fCamX
	invLenR := float32(1.0 / math.Sqrt(float64(rCamX*rCamX+rCamZ*rCamZ)))
	rCamX, rCamZ = rCamX*invLenR, rCamZ*invLenR

	uCamX := rCamY*fCamZ - rCamZ*fCamY
	uCamY := rCamZ*fCamX - rCamX*fCamZ
	uCamZ := rCamX*fCamY - rCamY*fCamX

	// Posizionamento fisico originario della Torcia (Offset applicato)
	flashX := float32(camX) + (rCamX * flashOffsetX) + (uCamX * flashOffsetY)
	flashY := float32(camZ) + (rCamY * flashOffsetX) + (uCamY * flashOffsetY)
	flashZ := float32(-camY) + (rCamZ * flashOffsetX) + (uCamZ * flashOffsetY)

	// Crosshair virtuale a 512 unità per la convergenza del raggio
	targetX := float32(camX) + (fCamX * rayUnit)
	targetY := float32(camZ) + (fCamY * rayUnit)
	targetZ := float32(-camY) + (fCamZ * rayUnit)

	// Triangolazione: Il Forward punta dalla torcia verso il crosshair
	fX := targetX - flashX
	fY := targetY - flashY
	fZ := targetZ - flashZ
	invLenFlashF := float32(1.0 / math.Sqrt(float64(fX*fX+fY*fY+fZ*fZ)))
	fX, fY, fZ = fX*invLenFlashF, fY*invLenFlashF, fZ*invLenFlashF

	// Right = Forward x Up
	rX := fY*uCamZ - fZ*uCamY
	rY := fZ*uCamX - fX*uCamZ
	rZ := fX*uCamY - fY*uCamX
	invLenFlashR := float32(1.0 / math.Sqrt(float64(rX*rX+rY*rY+rZ*rZ)))
	rX, rY, rZ = rX*invLenFlashR, rY*invLenFlashR, rZ*invLenFlashR

	// Up = Right x Forward
	uX := rY*fZ - rZ*fY
	uY := rZ*fX - rX*fZ
	uZ := rX*fY - rY*fX

	tx := -(rX*flashX + rY*flashY + rZ*flashZ)
	ty := -(uX*flashX + uY*flashY + uZ*flashZ)
	tz := fX*flashX + fY*flashY + fZ*flashZ

	flashView := [16]float32{
		rX, uX, -fX, 0,
		rY, uY, -fY, 0,
		rZ, uZ, -fZ, 0,
		tx, ty, tz, 1,
	}

	flashSpace := MatrixMultiply4x4(_flashProj, flashView)
	return roomSpace, flashSpace
}

// MatrixMultiply4x4 multiplies two 4x4 matrices represented as 1D arrays and returns the resulting matrix.
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

// MatrixInverse4x4 computes the inverse of a 4x4 matrix `m` and returns the inverted matrix and a success flag.
// Returns `([16]float32{}, false)` if the matrix is not invertible.
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
