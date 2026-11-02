package metrics

import (
	"math"
)

// Frustum represents a viewing frustum in 3D space with projection and view matrices.
type Frustum struct {
	proj [16]float32
	view [16]float32
	fm   [16]float32

	projPtr *float32
	viewPtr *float32
	fmPtr   *float32
	scaleX  float32
	scaleY  float32
	p1      float32
	p2      float32
}

// NewFrustum creates and initializes a new Frustum instance and returns a pointer to it.
func NewFrustum() *Frustum {
	v := &Frustum{}
	v.projPtr = &v.proj[0]
	v.viewPtr = &v.view[0]
	v.fmPtr = &v.fm[0]
	return v
}

// Setup configures the frustum's projection scaling factors and plane parameters based on the provided aspect ratio and depth values.
func (f *Frustum) Setup(aspect, zFar, zNear float32, prjScaleX, prjScaleY float64) {
	const kScale = float32(1.0)
	f.scaleX = (kScale / aspect) * float32(prjScaleX)
	f.scaleY = kScale * float32(prjScaleY)
	f.p1 = (zFar + zNear) / (zNear - zFar)
	f.p2 = (2 * zFar * zNear) / (zNear - zFar)

	//if zFar <= zNear {
	//	zFar = 10000.0
	//}
}

// GetPtr returns a pointer to the final transformation matrix (`fm`) of the frustum.
func (f *Frustum) GetPtr() *float32 {
	return f.fmPtr
}

// Get returns the final transformation matrix of the frustum as a 4x4 float32 array.
func (f *Frustum) Get() [16]float32 {
	return f.fm
}

// Update recalculates the projection, view, and final transformation matrices for the frustum based on the provided parameters.
func (f *Frustum) Update(px, py, pz float32, yaw, pitch, roll float64) {
	f.proj[0], f.proj[1], f.proj[2], f.proj[3] = f.scaleX, 0, 0, 0
	f.proj[4], f.proj[5], f.proj[6], f.proj[7] = 0, f.scaleY, 0, 0
	f.proj[8], f.proj[9], f.proj[10], f.proj[11] = 0, 0, f.p1, -1
	f.proj[12], f.proj[13], f.proj[14], f.proj[15] = 0, 0, f.p2, 0
	// Trigonometria sferica pura per Z-UP
	cosY, sinY := float32(math.Cos(yaw)), float32(math.Sin(yaw))
	cosP, sinP := float32(math.Cos(pitch)), float32(math.Sin(pitch))
	cosR, sinR := float32(math.Cos(roll)), float32(math.Sin(roll))
	// Forward
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

	f.view[0], f.view[1], f.view[2], f.view[3] = rX, uX, -fX, 0
	f.view[4], f.view[5], f.view[6], f.view[7] = rY, uY, -fY, 0
	f.view[8], f.view[9], f.view[10], f.view[11] = rZ, uZ, -fZ, 0
	f.view[12], f.view[13], f.view[14], f.view[15] = tx, ty, tz, 1

	MatrixMultiply4x4Ptr(f.fmPtr, f.projPtr, f.viewPtr)
}
