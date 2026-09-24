package pixels

import (
	"fmt"
	"math"
)

// XY represents a 2D vector with X and Y components as float64 values.
type XY struct {
	X, Y float64
}

// ZV represents the zero vector, initialized as XY{0, 0}. It is commonly used as a default or empty XY value.
var ZV = XY{0, 0}

// MakeVec creates a XY instance with the specified X and Y coordinates.
func MakeVec(x float64, y float64) XY {
	return XY{x, y}
}

// nearlyEqual determines if two floating-point numbers are approximately equal within a specified precision threshold.
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

// Eq checks if the vector is nearly equal to another vector by comparing their components within a small tolerance.
func (u XY) Eq(v XY) bool {
	return nearlyEqual(u.X, v.X) && nearlyEqual(u.Y, v.Y)
}

// String returns the string representation of the XY in the format "Vec(X, Y)".
func (u XY) String() string {
	return fmt.Sprintf("Vec(%v, %v)", u.X, u.Y)
}

// XY returns the X and Y components of the vector as separate float64 values.
func (u XY) XY() (x, y float64) {
	return u.X, u.Y
}

// Add returns a new XY that is the component-wise sum of the receiver and the given XY.
func (u XY) Add(v XY) XY {
	return XY{
		u.X + v.X,
		u.Y + v.Y,
	}
}

// Sub subtracts the components of vector v from vector u and returns the resulting vector.
func (u XY) Sub(v XY) XY {
	return XY{
		u.X - v.X,
		u.Y - v.Y,
	}
}

// Floor rounds down the components of the vector to the nearest integers, returning a new XY with the floored values.
func (u XY) Floor() XY {
	return XY{
		math.Floor(u.X),
		math.Floor(u.Y),
	}
}

// To returns a new XY representing the vector from the current XY to the specified XY.
func (u XY) To(v XY) XY {
	return XY{
		v.X - u.X,
		v.Y - u.Y,
	}
}

// Scaled returns a new XY scaled by the given factor c.
func (u XY) Scaled(c float64) XY {
	return XY{u.X * c, u.Y * c}
}

// ScaledXY returns a new vector by component-wise scaling of the calling vector with the values of the input vector.
func (u XY) ScaledXY(v XY) XY {
	return XY{u.X * v.X, u.Y * v.Y}
}

// Len returns the Euclidean length (magnitude) of the vector.
func (u XY) Len() float64 {
	return math.Hypot(u.X, u.Y)
}

// Angle returns the angle of the vector in radians measured counterclockwise from the positive X-axis.
func (u XY) Angle() float64 {
	return math.Atan2(u.Y, u.X)
}

// Unit returns a unit vector in the same direction as the vector. For a zero vector, it returns XY{1, 0}.
func (u XY) Unit() XY {
	if u.X == 0 && u.Y == 0 {
		return XY{1, 0}
	}
	return u.Scaled(1 / u.Len())
}

// Rotated returns a new XY that is the original XY rotated by the given angle (in radians) around the origin.
func (u XY) Rotated(angle float64) XY {
	sin, cos := math.Sincos(angle)
	return XY{
		u.X*cos - u.Y*sin,
		u.X*sin + u.Y*cos,
	}
}

// Normal returns a vector perpendicular to the current vector, rotated 90 degrees counterclockwise.
func (u XY) Normal() XY {
	return XY{-u.Y, u.X}
}

// Dot calculates and returns the dot product of the current vector with another vector v.
func (u XY) Dot(v XY) float64 {
	return u.X*v.X + u.Y*v.Y
}

// Cross calculates the 2D cross product (determinant) of the calling vector and another vector.
func (u XY) Cross(v XY) float64 {
	return u.X*v.Y - v.X*u.Y
}

// Project computes the vector projection of the current vector onto the given vector v and returns the resulting vector.
func (u XY) Project(v XY) XY {
	lenV := u.Dot(v) / v.Len()
	return v.Unit().Scaled(lenV)
}

// Map applies a given function to both X and Y components of the vector and returns a new vector with the results.
func (u XY) Map(f func(float64) float64) XY {
	return XY{
		f(u.X),
		f(u.Y),
	}
}

// Lerp linearly interpolates between two vectors `a` and `b` by a factor `t`, returning the resulting vector.
func Lerp(a, b XY, t float64) XY {
	return a.Scaled(1 - t).Add(b.Scaled(t))
}

// Unit returns a unit vector with its direction determined by the provided angle in radians.
func Unit(angle float64) XY {
	return XY{1, 0}.Rotated(angle)
}
