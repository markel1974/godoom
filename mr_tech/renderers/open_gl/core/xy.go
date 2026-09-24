package core

import (
	"fmt"
	"math"
)

// XY represents a 2D vector or point with X and Y as float64 components.
type XY struct {
	X, Y float64
}

// ZV is the zero vector of type XY with both X and Y set to 0. Used as a default or uninitialized XY value.
var ZV = XY{0, 0}

// MakeVec creates a new XY vector with the specified x and y float64 components.
func MakeVec(x float64, y float64) XY {
	return XY{x, y}
}

// Eq checks if two XY vectors are approximately equal by comparing their respective components within a tolerance threshold.
func (u XY) Eq(v XY) bool {
	return nearlyEqual(u.X, v.X) && nearlyEqual(u.Y, v.Y)
}

// String returns a string representation of the XY struct in the format "Vec(X, Y)".
func (u XY) String() string {
	return fmt.Sprintf("Vec(%v, %v)", u.X, u.Y)
}

// XY returns the X and Y coordinates of the XY struct as separate float64 values.
func (u XY) XY() (x, y float64) {
	return u.X, u.Y
}

// Add returns a new XY vector that is the sum of the current vector and the given vector.
func (u XY) Add(v XY) XY {
	return XY{
		u.X + v.X,
		u.Y + v.Y,
	}
}

// Sub subtracts the components of vector v from the components of vector u and returns the resulting vector.
func (u XY) Sub(v XY) XY {
	return XY{
		u.X - v.X,
		u.Y - v.Y,
	}
}

// Floor returns a new XY where each component is the largest integer less than or equal to the corresponding component.
func (u XY) Floor() XY {
	return XY{
		math.Floor(u.X),
		math.Floor(u.Y),
	}
}

// To computes the vector difference between the current XY and the provided XY and returns the resulting XY.
func (u XY) To(v XY) XY {
	return XY{
		v.X - u.X,
		v.Y - u.Y,
	}
}

// Scaled multiplies both components of the XY vector by the given scalar value and returns the resulting vector.
func (u XY) Scaled(c float64) XY {
	return XY{u.X * c, u.Y * c}
}

// ScaledXY returns a new XY with its components scaled by the respective components of the input vector v.
func (u XY) ScaledXY(v XY) XY {
	return XY{u.X * v.X, u.Y * v.Y}
}

// Len computes and returns the length (magnitude) of the vector.
func (u XY) Len() float64 {
	return math.Hypot(u.X, u.Y)
}

// Angle computes the angle in radians between the vector and the positive X-axis, measured counterclockwise.
func (u XY) Angle() float64 {
	return math.Atan2(u.Y, u.X)
}

// Unit returns a unit vector in the same direction as the current vector. Defaults to {1, 0} if the vector is zero.
func (u XY) Unit() XY {
	if u.X == 0 && u.Y == 0 {
		return XY{1, 0}
	}
	return u.Scaled(1 / u.Len())
}

// Rotated rotates the XY vector by the specified angle (in radians) and returns the resulting new XY vector.
func (u XY) Rotated(angle float64) XY {
	sin, cos := math.Sincos(angle)
	return XY{
		u.X*cos - u.Y*sin,
		u.X*sin + u.Y*cos,
	}
}

// Normal returns a new XY vector that is the normal (perpendicular) to the original vector, rotated 90 degrees counterclockwise.
func (u XY) Normal() XY {
	return XY{-u.Y, u.X}
}

// Dot computes the dot product of the vector u and vector v as a float64.
func (u XY) Dot(v XY) float64 {
	return u.X*v.X + u.Y*v.Y
}

// Cross calculates the 2D cross product of two vectors and returns the scalar result.
func (u XY) Cross(v XY) float64 {
	return u.X*v.Y - v.X*u.Y
}

// Project computes the projection of the vector u onto the vector v and returns the resulting vector.
func (u XY) Project(v XY) XY {
	lenV := u.Dot(v) / v.Len()
	return v.Unit().Scaled(lenV)
}

// Map applies a given function to both X and Y components of the XY struct and returns a new XY with transformed values.
func (u XY) Map(f func(float64) float64) XY {
	return XY{
		f(u.X),
		f(u.Y),
	}
}

// nearlyEqual determines if two floating-point numbers are approximately equal within a small tolerance (epsilon).
func nearlyEqual(a float64, b float64) bool {
	epsilon := 0.000001

	if a == b {
		return true
	}

	diff := math.Abs(a - b)

	if a == 0.0 || b == 0.0 || diff < math.SmallestNonzeroFloat64 {
		return diff < (epsilon * math.SmallestNonzeroFloat64)
	}

	absA := math.Abs(a)
	absB := math.Abs(b)

	return diff/min(absA+absB, math.MaxFloat64) < epsilon
}

// Lerp performs linear interpolation between two XY points a and b using the parameter t and returns the result.
func Lerp(a, b XY, t float64) XY {
	return a.Scaled(1 - t).Add(b.Scaled(t))
}

// Unit returns a unit vector rotated by the given angle (in radians) counterclockwise from the positive X-axis.
func Unit(angle float64) XY {
	return XY{1, 0}.Rotated(angle)
}
