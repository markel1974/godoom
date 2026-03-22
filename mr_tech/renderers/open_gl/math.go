package open_gl

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model"
)

// CreateSpaces generates and returns the room and flash projection-view matrices used for rendering transformations.
func CreateSpaces(vi *model.ViewMatrix, pX, pY float64, flashOffsetX, flashOffsetY float32) ([16]float32, [16]float32) {
	// --- 1. PROIEZIONE STANZA (Ortografica) ---
	const orthoSize = 1024.0
	const zNearRoom, zFarRoom = 1.0, 4096.0
	const zNearFlash, zFarFlash = 0.1, 2048.0

	roomProj := [16]float32{
		1.0 / orthoSize, 0, 0, 0,
		0, 1.0 / orthoSize, 0, 0,
		0, 0, -2.0 / (zFarRoom - zNearRoom), 0,
		0, 0, -(zFarRoom + zNearRoom) / (zFarRoom - zNearRoom), 1,
	}

	texelSize := (orthoSize * 2.0) / 1024.0
	snappedX := math.Floor(pX/texelSize) * texelSize
	snappedY := math.Floor(-pY/texelSize) * texelSize

	lX, lY, lZ := float32(snappedX), float32(1024.0), float32(snappedY)

	roomView := [16]float32{
		1, 0, 0, 0,
		0, 0, 1, 0,
		0, -1, 0, 0,
		-lX, lZ, -lY, 1,
	}

	var roomSpace [16]float32
	MultiplyMatrix(&roomSpace, roomProj, roomView)

	// 2. SINCRONIZZAZIONE FOV TORCIA (Avvolge i 106° calcolati dallo shader)
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

	//fX, fY, fZ := float32(cosA), pitchShear, float32(-sinA)
	//invLenF := float32(1.0 / math.Sqrt(float64(fX*fX+fY*fY+fZ*fZ)))
	//fX, fY, fZ = fX*invLenF, fY*invLenF, fZ*invLenF
	scaleY := float32(2.0 * model.VFov)
	flashDirY := pitchShear / scaleY

	// Usa flashDirY al posto di pitchShear puro
	fX, fY, fZ := float32(cosA), flashDirY, float32(-sinA)
	invLenF := float32(1.0 / math.Sqrt(float64(fX*fX+fY*fY+fZ*fZ)))
	fX, fY, fZ = fX*invLenF, fY*invLenF, fZ*invLenF

	rX, rY, rZ := -fZ, float32(0.0), fX
	invLenR := float32(1.0 / math.Sqrt(float64(rX*rX+rZ*rZ)))
	rX, rZ = rX*invLenR, rZ*invLenR

	uX := rY*fZ - rZ*fY
	uY := rZ*fX - rX*fZ
	uZ := rX*fY - rY*fX

	flashX := float32(camX) + (rX * flashOffsetX) + (uX * flashOffsetY)
	flashY := float32(camZ) + (rY * flashOffsetX) + (uY * flashOffsetY)
	flashZ := float32(-camY) + (rZ * flashOffsetX) + (uZ * flashOffsetY)

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
