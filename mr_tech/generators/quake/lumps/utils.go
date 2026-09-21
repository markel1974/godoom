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
	parts := strings.Fields(s)
	if len(parts) >= 3 {
		v1, _ := strconv.ParseFloat(parts[0], 64)
		v2, _ := strconv.ParseFloat(parts[1], 64)
		v3, _ := strconv.ParseFloat(parts[2], 64)
		return v1, v2, v3, true
	}
	return 0, 0, 0, false
}

// CalcDirection converts Quake angles (yaw, pitch) into a normalized direction vector.
func CalcDirection(yaw, pitch float64) (float64, float64, float64) {
	yawRad := yaw * math.Pi / 180.0
	pitchRad := pitch * math.Pi / 180.0
	dirX := math.Cos(pitchRad) * math.Cos(yawRad)
	dirY := math.Sin(pitchRad)
	dirZ := math.Cos(pitchRad) * math.Sin(yawRad)
	return dirX, dirY, dirZ
}
