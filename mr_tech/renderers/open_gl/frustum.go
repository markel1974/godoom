package open_gl

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model"
)

// CreateFrontRearFrustum generates view-projection matrices for front and rear frustums based on given camera parameters.
func CreateFrontRearFrustum(aspect, zFarRoom, px, py, pz float32, angle, pitch, roll float64) ([16]float32, [16]float32) {
	rAngle, rPitch, rRoll := angle+math.Pi, -pitch, -roll
	fm := CreateFrustum(aspect, zFarRoom, px, py, pz, angle, pitch, roll)
	fr := CreateFrustum(aspect, zFarRoom, px, py, pz, rAngle, rPitch, rRoll)
	return fm, fr
}

// CreateFrustum generates a view-projection matrix for rendering a 3D frustum with specified dimensions and orientation.
func CreateFrustum(aspect, zFarRoom, px, py, pz float32, yaw, pitch, roll float64) [16]float32 {
	//aspect := fbw / fbh
	scaleX := (2.0 / aspect) * float32(model.HFov)
	scaleY := 2.0 * float32(model.VFov)
	zNear := float32(0.1)
	zFar := zFarRoom
	if zFar <= zNear {
		zFar = 10000.0
	}
	proj := [16]float32{
		scaleX, 0, 0, 0,
		0, scaleY, 0, 0,
		0, 0, (zFar + zNear) / (zNear - zFar), -1,
		0, 0, (2 * zFar * zNear) / (zNear - zFar), 0,
	}
	// Trigonometria sferica pura per Z-UP
	cosY, sinY := float32(math.Cos(yaw)), float32(math.Sin(yaw))
	cosP, sinP := float32(math.Cos(pitch)), float32(math.Sin(pitch))
	cosR, sinR := float32(math.Cos(roll)), float32(math.Sin(roll))
	// Forward (basato sul calcolo tz originale del motore 2D, esteso in 3D)
	fX := cosY * cosP
	fY := sinY * cosP
	fZ := sinP
	// Right base (Yaw - 90 deg)
	rXb := sinY
	rYb := -cosY
	// Up base (ortogonale al Pitch)
	uXb := -cosY * sinP
	uYb := -sinY * sinP
	uZb := cosP
	// Applicazione del Roll ai vettori Right e Up
	rX := rXb*cosR + uXb*sinR
	rY := rYb*cosR + uYb*sinR
	rZ := uZb * sinR
	uX := uXb*cosR - rXb*sinR
	uY := uYb*cosR - rYb*sinR
	uZ := uZb * cosR
	// Traslazione View Space (prodotto scalare negativo contro gli assi estratti)
	tx := -(rX*px + rY*py + rZ*pz)
	ty := -(uX*px + uY*py + uZ*pz)
	tz := fX*px + fY*py + fZ*pz
	view := [16]float32{
		rX, uX, -fX, 0,
		rY, uY, -fY, 0,
		rZ, uZ, -fZ, 0,
		tx, ty, tz, 1,
	}
	vp := MatrixMultiply(proj, view)
	return vp
}

// MatrixMultiply multiplies two 4x4 matrices `proj` and `view`, returning the resulting 4x4 matrix as a 1D array.
func MatrixMultiply(proj, view [16]float32) [16]float32 {
	var vp [16]float32
	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
			sum := float32(0.0)
			for i := 0; i < 4; i++ {
				sum += proj[i*4+row] * view[col*4+i]
			}
			vp[col*4+row] = sum
		}
	}
	return vp
}
