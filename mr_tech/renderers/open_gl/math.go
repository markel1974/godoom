package open_gl

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model"
)

// CreateSpaces generates and returns the room and flash projection-view matrices used for rendering transformations.
// CreateSpaces generates and returns the room and flash projection-view matrices used for rendering transformations.
func CreateSpaces(vi *model.ViewMatrix, pX, pY float64, flashOffsetX, flashOffsetY float32) ([16]float32, [16]float32) {
	const zNearFlash, zFarFlash = 1.0, 2048.0

	// --- 1. PROIEZIONE STANZA (Ortografica Direzionale) ---
	const orthoSize = 640.0
	const zNearRoom, zFarRoom = 1.0, 8192.0

	roomProj := [16]float32{
		1.0 / orthoSize, 0, 0, 0,
		0, 1.0 / orthoSize, 0, 0,
		0, 0, -2.0 / (zFarRoom - zNearRoom), 0,
		0, 0, -(zFarRoom + zNearRoom) / (zFarRoom - zNearRoom), 1,
	}

	texelSize := (orthoSize * 2.0) / 1024.0

	snappedX := math.Floor(pX/texelSize) * texelSize
	snappedY := math.Floor(-pY/texelSize) * texelSize

	// lY a 4096.0 impedisce il near-plane clipping dei soffitti
	lX, lY, lZ := float32(snappedX), float32(4096.0), float32(snappedY) // Telecamera altissima per non tagliare i soffitti
	roomView := [16]float32{
		1, 0, 0, 0,
		0.02, 0.02, 1, 0, // TILT: Inclinazione impercettibile per dare area d'ombra ai muri
		0, -1, 0, 0,
		-lX, lZ, -lY, 1,
	}

	var roomSpace [16]float32
	MultiplyMatrix(&roomSpace, roomProj, roomView)

	// --- 2. SINCRONIZZAZIONE FOV E PARALLASSE TORCIA (Prospettica) ---
	fovRad := float32(110.0 * math.Pi / 180.0)
	f := float32(1.0 / math.Tan(float64(fovRad/2.0)))

	flashProj := [16]float32{
		f, 0, 0, 0,
		0, f, 0, 0,
		0, 0, (zFarFlash + zNearFlash) / (zNearFlash - zFarFlash), -1,
		0, 0, (2 * zFarFlash * zNearFlash) / (zNearFlash - zFarFlash), 0,
	}

	sinA, cosA := vi.GetAngle()
	camX, camY := vi.GetXY()
	camZ := vi.GetZ()
	pitchShear := float32(-vi.GetYaw())

	scaleY := float32(2.0 * model.VFov)
	flashDirY := pitchShear / scaleY

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
	targetX := float32(camX) + (fCamX * 512.0)
	targetY := float32(camZ) + (fCamY * 512.0)
	targetZ := float32(-camY) + (fCamZ * 512.0)

	// Triangolazione: Il Forward punta dalla torcia verso il crosshair
	fX := targetX - flashX
	fY := targetY - flashY
	fZ := targetZ - flashZ
	invLenFlashF := float32(1.0 / math.Sqrt(float64(fX*fX+fY*fY+fZ*fZ)))
	fX, fY, fZ = fX*invLenFlashF, fY*invLenFlashF, fZ*invLenFlashF

	// Ricalcolo ortonormale (Right/Up) allineato al nuovo Forward
	rX := uCamY*fZ - uCamZ*fY
	rY := uCamZ*fX - uCamX*fZ
	rZ := uCamX*fY - uCamY*fX
	invLenFlashR := float32(1.0 / math.Sqrt(float64(rX*rX+rY*rY+rZ*rZ)))
	rX, rY, rZ = rX*invLenFlashR, rY*invLenFlashR, rZ*invLenFlashR

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

	var flashSpace [16]float32
	MultiplyMatrix(&flashSpace, flashProj, flashView)

	return roomSpace, flashSpace
}

// MultiplyMatrix multiplies two 4x4 matrices `a` and `b`, storing the result in `out`.
func MultiplyMatrix(out *[16]float32, a [16]float32, b [16]float32) {
	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
			sum := float32(0.0)
			for i := 0; i < 4; i++ {
				sum += a[i*4+row] * b[col*4+i]
			}
			out[col*4+row] = sum
		}
	}
}
