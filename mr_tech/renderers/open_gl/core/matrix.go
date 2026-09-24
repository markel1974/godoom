package core

import (
	"fmt"
	"math"
)

// Matrix represents a 2D affine transformation matrix stored as a flat array of 6 float64 values.
type Matrix [6]float64

// IM represents the identity matrix in 2D affine transformations, preserving position, scale, and rotation.
var IM = Matrix{1, 0, 0, 1, 0, 0}

// String returns a string representation of the Matrix in a human-readable format.
func (m Matrix) String() string {
	return fmt.Sprintf(
		"Matrix(%v %v %v | %v %v %v)",
		m[0], m[2], m[4],
		m[1], m[3], m[5],
	)
}

// Moved returns a new Matrix by translating the current Matrix by the specified delta vector in 2D space.
func (m Matrix) Moved(delta XY) Matrix {
	m[4], m[5] = m[4]+delta.X, m[5]+delta.Y
	return m
}

// ScaledXY applies a scaling transformation to the matrix around a given point with independent X and Y scale factors.
func (m Matrix) ScaledXY(around XY, scale XY) Matrix {
	m[4], m[5] = m[4]-around.X, m[5]-around.Y
	m[0], m[2], m[4] = m[0]*scale.X, m[2]*scale.X, m[4]*scale.X
	m[1], m[3], m[5] = m[1]*scale.Y, m[3]*scale.Y, m[5]*scale.Y
	m[4], m[5] = m[4]+around.X, m[5]+around.Y
	return m
}

// Scaled returns a new Matrix scaled uniformly by the given scale factor around the specified point.
func (m Matrix) Scaled(around XY, scale float64) Matrix {
	return m.ScaledXY(around, MakeVec(scale, scale))
}

// Rotated returns a new Matrix that is the result of rotating the current Matrix around the given point by the specified angle.
func (m Matrix) Rotated(around XY, angle float64) Matrix {
	sinT, cosT := math.Sincos(angle)
	m[4], m[5] = m[4]-around.X, m[5]-around.Y
	m2 := m.Chained(Matrix{cosT, sinT, -sinT, cosT, 0, 0})
	m2[4], m2[5] = m2[4]+around.X, m2[5]+around.Y
	return m2
}

// Chained multiplies the current matrix with another matrix and returns the resulting transformation matrix.
func (m Matrix) Chained(next Matrix) Matrix {
	return Matrix{
		next[0]*m[0] + next[2]*m[1],
		next[1]*m[0] + next[3]*m[1],
		next[0]*m[2] + next[2]*m[3],
		next[1]*m[2] + next[3]*m[3],
		next[0]*m[4] + next[2]*m[5] + next[4],
		next[1]*m[4] + next[3]*m[5] + next[5],
	}
}

// Project applies the transformation defined by the Matrix to the given XY point and returns the resulting XY.
func (m Matrix) Project(u XY) XY {
	return XY{m[0]*u.X + m[2]*u.Y + m[4], m[1]*u.X + m[3]*u.Y + m[5]}
}

// Unproject applies the inverse of the affine transformation matrix to the given 2D point and returns the resulting point.
func (m Matrix) Unproject(u XY) XY {
	det := m[0]*m[3] - m[2]*m[1]
	return XY{
		(m[3]*(u.X-m[4]) - m[2]*(u.Y-m[5])) / det,
		(-m[1]*(u.X-m[4]) + m[0]*(u.Y-m[5])) / det,
	}
}
