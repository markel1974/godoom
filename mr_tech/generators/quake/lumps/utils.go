package lumps

import (
	"errors"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/geometry"
)

// ToString converts an array of 8 bytes into a string, stopping at the first null byte or the end of the array.
func ToString(s [8]byte) string {
	var i int
	for i = 0; i < len(s); i++ {
		if s[i] == 0 {
			break
		}
	}
	return string(s[:i])
}

// Seek moves the file pointer of the provided file to the specified offset relative to the start of the file.
// Returns an error if the seek operation fails or if the resulting position does not match the requested offset.
func Seek(reader io.ReadSeeker, offset int64) error {
	//off, err := f.Seek(offset, os.SEEK_SET)
	off, err := reader.Seek(offset, io.SeekStart)
	if err != nil {
		return err
	}
	if off != offset {
		return errors.New("seek failed")
	}
	return nil
}

// FixName normalizes the input string by converting it to uppercase and trimming whitespace and control characters.
func FixName(in string) string {
	return strings.Trim(strings.ToUpper(in), "\n\r\t ")
}

func FromNullTerminatingString(in []byte) string {
	var out []rune
	for _, i := range in {
		if i == 0 {
			break
		}
		out = append(out, rune(i))
	}
	return string(out)
}

// TriangulateConvex3d generates a triangle fan from a convex 3D polygon defined by a list of vertices.
// It returns a slice of slices, each containing exactly three vertices representing a single triangle.
func TriangulateConvex3d(pts []geometry.XYZ) [][]geometry.XYZ {
	pLen := len(pts)
	if pLen < 3 {
		return nil // Degenerate polygon
	}
	if pLen == 3 {
		return [][]geometry.XYZ{{pts[0], pts[1], pts[2]}}
	}
	output := make([][]geometry.XYZ, 0, pLen-2)
	// Triangle Fan anchored to pts[0]
	for i := 1; i < pLen-1; i++ {
		output = append(output, []geometry.XYZ{pts[0], pts[i], pts[i+1]})
	}
	return output
}

// TriangulateConvex3dInverted triangulates a convex 3D polygon into triangles in inverted winding order.
func TriangulateConvex3dInverted(pts []geometry.XYZ) [][]geometry.XYZ {
	pLen := len(pts)
	if pLen < 3 {
		return nil
	}
	if pLen == 3 {
		// INVERTED: from (0, 1, 2) to (0, 2, 1)
		return [][]geometry.XYZ{{pts[0], pts[2], pts[1]}}
	}

	output := make([][]geometry.XYZ, 0, pLen-2)
	for i := 1; i < pLen-1; i++ {
		// INVERTED: pts[i+1] comes BEFORE pts[i]
		output = append(output, []geometry.XYZ{pts[0], pts[i+1], pts[i]})
	}
	return output
}

// ParseVector extracts 3 floats from a Quake-style string (e.g. "1.0 0.5 0.0").
func ParseVector(s string) (float64, float64, float64, bool) {
	if len(s) == 0 {
		return 0, 0, 0, false
	}
	parts := strings.Fields(s)
	if len(parts) >= 3 {
		v1, _ := strconv.ParseFloat(parts[0], 64)
		v2, _ := strconv.ParseFloat(parts[1], 64)
		v3, _ := strconv.ParseFloat(parts[2], 64)
		return v1, v2, v3, true
	}
	return 0, 0, 0, false
}

// ParseFloat converts a string to a float64 and returns whether the conversion was successful.
func ParseFloat(s string) (float64, bool) {
	if len(s) == 0 {
		return 0, false
	}
	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

// ParseColorVector parses a color string, normalizes its components, and returns them along with a validity flag.
func ParseColorVector(colorStr string) (float64, float64, float64, bool) {
	if len(colorStr) == 0 {
		return 0, 0, 0, false
	}
	cr, cg, cb, valid := ParseVector(colorStr)
	if !valid {
		return 0, 0, 0, false
	}
	var r, g, b float64
	if cr > 1.0 || cg > 1.0 || cb > 1.0 {
		r = cr / 255.0
		g = cg / 255.0
		b = cb / 255.0
	} else {
		r = cr
		g = cg
		b = cb
	}
	return r, g, b, true
}

// CalcDirectionYUP calculates a 3D directional vector (X, Y, Z) in a Y-up coordinate system based on yaw and pitch angles.
func CalcDirectionYUP(yaw, pitch float64) (float64, float64, float64) {
	yawRad := yaw * math.Pi / 180.0
	pitchRad := pitch * math.Pi / 180.0

	cp := math.Cos(pitchRad)
	sp := math.Sin(pitchRad)
	cy := math.Cos(yawRad)
	sy := math.Sin(yawRad)

	dirX := cp * cy
	dirY := sp
	dirZ := cp * sy
	return dirX, dirY, dirZ
}

// CalcDirectionZUP calculates a directional vector in a Z-up coordinate system given yaw and pitch angles in degrees.
// Returns the X, Y, and Z components of the vector as float64 values.
func CalcDirectionZUP(yaw, pitch float64) (float64, float64, float64) {
	yawRad := yaw * math.Pi / 180.0
	pitchRad := pitch * math.Pi / 180.0
	cp := math.Cos(pitchRad)
	sp := math.Sin(pitchRad)
	cy := math.Cos(yawRad)
	sy := math.Sin(yawRad)
	dirX := sp * cy
	dirY := sp * sy
	dirZ := -cp
	return dirX, dirY, dirZ
}

// CalcDirection calculates the X, Y, and Z components of a direction vector based on yaw and pitch angles in degrees.
func CalcDirection(yaw, pitch float64) (float64, float64, float64) {
	dirX, dirY, dirZ := CalcDirectionZUP(yaw, pitch)
	return dirX, dirY, dirZ
}

// CalcAngleDirectionZUP calculates the direction in the XY-plane for a given angle assuming Z-axis is up (Z-UP).
// The angle is provided in degrees and the function returns the X, Y, and Z components of the direction vector.
func CalcAngleDirectionZUP(angle float64) (float64, float64, float64) {
	angleRad := angle * math.Pi / 180.0

	dirX := math.Cos(angleRad)
	dirY := math.Sin(angleRad)
	dirZ := 0.0

	return dirX, dirY, dirZ
}

// CalcAngleDirection computes the direction components (X, Y, Z) for a given angle in degrees.
func CalcAngleDirection(angle float64) (float64, float64, float64) {
	dirX, dirY, dirZ := CalcAngleDirectionZUP(angle)
	return dirX, dirY, dirZ
}

// CreateXYZ creates an XYZ struct with specified X, Y, and Z coordinates.
func CreateXYZ(x, y, z float64) geometry.XYZ {
	// Conversione coordinate: Quake Z-up -> Engine Z-up
	//pos := geometry.XYZ{X: x, Y: z, Z: -y}
	pos := geometry.XYZ{X: x, Y: y, Z: z}
	return pos
}
