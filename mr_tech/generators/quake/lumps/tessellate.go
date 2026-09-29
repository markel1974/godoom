package lumps

import (
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// EvalBezier evaluates a quadratic Bezier curve at a given parameter t (0 <= t <= 1) using control points p0, p1, and p2.
func EvalBezier(p0, p1, p2 float32, t float32) float32 {
	u := 1.0 - t
	return (u * u * p0) + (2.0 * u * t * p1) + (t * t * p2)
}

// Tessellate generates a tessellated surface and its UV coordinates from a 3x3 Bezier patch and subdivision level.
func Tessellate(cp [9]Vertex3, level int) ([]geometry.XYZ, [][2]float64) {
	var points []geometry.XYZ
	var uvs [][2]float64
	step := 1.0 / float32(level)
	L := level + 1
	grid := make([]geometry.XYZ, L*L)
	gridUV := make([][2]float64, L*L)

	// Calcolo interpolazione griglia
	for i := 0; i <= level; i++ {
		tV := float32(i) * step
		for j := 0; j <= level; j++ {
			tU := float32(j) * step
			var p [3]geometry.XYZ
			var puv [3][2]float32
			for row := 0; row < 3; row++ {
				idx := row * 3
				p[row] = CreateXYZ(
					float64(EvalBezier(cp[idx].Position[0], cp[idx+1].Position[0], cp[idx+2].Position[0], tU)),
					float64(EvalBezier(cp[idx].Position[1], cp[idx+1].Position[1], cp[idx+2].Position[1], tU)),
					float64(EvalBezier(cp[idx].Position[2], cp[idx+1].Position[2], cp[idx+2].Position[2], tU)),
				)
				puv[row] = [2]float32{
					EvalBezier(cp[idx].TexCoord[0], cp[idx+1].TexCoord[0], cp[idx+2].TexCoord[0], tU),
					EvalBezier(cp[idx].TexCoord[1], cp[idx+1].TexCoord[1], cp[idx+2].TexCoord[1], tU),
				}
			}
			grid[i*L+j] = CreateXYZ(
				float64(EvalBezier(float32(p[0].X), float32(p[1].X), float32(p[2].X), tV)),
				float64(EvalBezier(float32(p[0].Y), float32(p[1].Y), float32(p[2].Y), tV)),
				float64(EvalBezier(float32(p[0].Z), float32(p[1].Z), float32(p[2].Z), tV)),
			)
			gridUV[i*L+j] = [2]float64{
				float64(EvalBezier(puv[0][0], puv[1][0], puv[2][0], tV)),
				float64(EvalBezier(puv[0][1], puv[1][1], puv[2][1], tV)),
			}
		}
	}

	// Chiusura dei quadrati in triangoli (Winding Order CCW)
	for i := 0; i < level; i++ {
		for j := 0; j < level; j++ {
			idx0 := (i * L) + j
			idx1 := (i * L) + j + 1
			idx2 := ((i + 1) * L) + j
			idx3 := ((i + 1) * L) + j + 1

			points = append(points, grid[idx0], grid[idx2], grid[idx1])
			uvs = append(uvs, gridUV[idx0], gridUV[idx2], gridUV[idx1])

			points = append(points, grid[idx1], grid[idx2], grid[idx3])
			uvs = append(uvs, gridUV[idx1], gridUV[idx2], gridUV[idx3])
		}
	}
	return points, uvs
}
