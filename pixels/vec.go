package pixels

import (
	"fmt"
	"math"
)

// Vec represents a 2D vector with X and Y components as float64 values.
type Vec struct {
	X, Y float64
}

// ZV represents the zero vector, initialized as Vec{0, 0}. It is commonly used as a default or empty Vec value.
var ZV = Vec{0, 0}

// MakeVec creates a Vec instance with the specified X and Y coordinates.
func MakeVec(x float64, y float64) Vec {
	return Vec{x, y}
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
func (u Vec) Eq(v Vec) bool {
	return nearlyEqual(u.X, v.X) && nearlyEqual(u.Y, v.Y)
}

// String returns the string representation of the Vec in the format "Vec(X, Y)".
func (u Vec) String() string {
	return fmt.Sprintf("Vec(%v, %v)", u.X, u.Y)
}

// XY returns the X and Y components of the vector as separate float64 values.
func (u Vec) XY() (x, y float64) {
	return u.X, u.Y
}

// Add returns a new Vec that is the component-wise sum of the receiver and the given Vec.
func (u Vec) Add(v Vec) Vec {
	return Vec{
		u.X + v.X,
		u.Y + v.Y,
	}
}

// Sub subtracts the components of vector v from vector u and returns the resulting vector.
func (u Vec) Sub(v Vec) Vec {
	return Vec{
		u.X - v.X,
		u.Y - v.Y,
	}
}

// Floor rounds down the components of the vector to the nearest integers, returning a new Vec with the floored values.
func (u Vec) Floor() Vec {
	return Vec{
		math.Floor(u.X),
		math.Floor(u.Y),
	}
}

// To returns a new Vec representing the vector from the current Vec to the specified Vec.
func (u Vec) To(v Vec) Vec {
	return Vec{
		v.X - u.X,
		v.Y - u.Y,
	}
}

// Scaled returns a new Vec scaled by the given factor c.
func (u Vec) Scaled(c float64) Vec {
	return Vec{u.X * c, u.Y * c}
}

// ScaledXY returns a new vector by component-wise scaling of the calling vector with the values of the input vector.
func (u Vec) ScaledXY(v Vec) Vec {
	return Vec{u.X * v.X, u.Y * v.Y}
}

// Len returns the Euclidean length (magnitude) of the vector.
func (u Vec) Len() float64 {
	return math.Hypot(u.X, u.Y)
}

// Angle returns the angle of the vector in radians measured counterclockwise from the positive X-axis.
func (u Vec) Angle() float64 {
	return math.Atan2(u.Y, u.X)
}

// Unit returns a unit vector in the same direction as the vector. For a zero vector, it returns Vec{1, 0}.
func (u Vec) Unit() Vec {
	if u.X == 0 && u.Y == 0 {
		return Vec{1, 0}
	}
	return u.Scaled(1 / u.Len())
}

// Rotated returns a new Vec that is the original Vec rotated by the given angle (in radians) around the origin.
func (u Vec) Rotated(angle float64) Vec {
	sin, cos := math.Sincos(angle)
	return Vec{
		u.X*cos - u.Y*sin,
		u.X*sin + u.Y*cos,
	}
}

// Normal returns a vector perpendicular to the current vector, rotated 90 degrees counterclockwise.
func (u Vec) Normal() Vec {
	return Vec{-u.Y, u.X}
}

// Dot calculates and returns the dot product of the current vector with another vector v.
func (u Vec) Dot(v Vec) float64 {
	return u.X*v.X + u.Y*v.Y
}

// Cross calculates the 2D cross product (determinant) of the calling vector and another vector.
func (u Vec) Cross(v Vec) float64 {
	return u.X*v.Y - v.X*u.Y
}

// Project computes the vector projection of the current vector onto the given vector v and returns the resulting vector.
func (u Vec) Project(v Vec) Vec {
	lenV := u.Dot(v) / v.Len()
	return v.Unit().Scaled(lenV)
}

// Map applies a given function to both X and Y components of the vector and returns a new vector with the results.
func (u Vec) Map(f func(float64) float64) Vec {
	return Vec{
		f(u.X),
		f(u.Y),
	}
}

// Lerp linearly interpolates between two vectors `a` and `b` by a factor `t`, returning the resulting vector.
func Lerp(a, b Vec, t float64) Vec {
	return a.Scaled(1 - t).Add(b.Scaled(t))
}

// Unit returns a unit vector with its direction determined by the provided angle in radians.
func Unit(angle float64) Vec {
	return Vec{1, 0}.Rotated(angle)
}
