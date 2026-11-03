package doom3

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// md5Joint holds the position and orientation of a single bone in model space.
type md5Joint struct {
	name   string
	parent int
	pos    geometry.XYZ
	orient [4]float64 // (qx, qy, qz, qw)
}

// md5Weight represents a vertex skinning weight attached to a specific joint.
type md5Weight struct {
	joint int
	bias  float64
	pos   geometry.XYZ
}

// md5Vert represents a vertex definition in the MD5 mesh.
type md5Vert struct {
	u          float32
	v          float32
	weightIdx  int
	weightSpan int
}

// quatRotate rotates a 3D vector by a unit quaternion (qx, qy, qz, qw).
func quatRotate(q [4]float64, v geometry.XYZ) geometry.XYZ {
	qx, qy, qz, qw := q[0], q[1], q[2], q[3]

	tx := 2.0 * (qy*v.Z - qz*v.Y)
	ty := 2.0 * (qz*v.X - qx*v.Z)
	tz := 2.0 * (qx*v.Y - qy*v.X)

	rx := v.X + qw*tx + (qy*tz - qz*ty)
	ry := v.Y + qw*ty + (qz*tx - qx*tz)
	rz := v.Z + qw*tz + (qx*ty - qy*tx)

	return geometry.XYZ{X: rx, Y: ry, Z: rz}
}

// ParseMD5Mesh parses a Doom 3 .md5mesh file and converts it into a Config Model3DEntry.
func ParseMD5Mesh(r io.Reader, materials *Materials) (*config.Model3DEntry, error) {
	scanner := bufio.NewScanner(r)

	var joints []md5Joint
	var allTriangles []config.Model3DEntryTriangle

	inJoints := false
	inMesh := false

	var currentShader string
	var currentVerts []md5Vert
	var currentWeights []md5Weight
	var currentTris [][3]int

	finishMesh := func() {
		if len(currentVerts) == 0 || len(currentTris) == 0 {
			return
		}

		// Calculate bind-pose vertex positions
		computedVerts := make([]geometry.XYZ, len(currentVerts))
		for i, vert := range currentVerts {
			var p geometry.XYZ
			for w := 0; w < vert.weightSpan; w++ {
				weight := currentWeights[vert.weightIdx+w]
				joint := joints[weight.joint]

				rotated := quatRotate(joint.orient, weight.pos)
				p.X += (joint.pos.X + rotated.X) * weight.bias
				p.Y += (joint.pos.Y + rotated.Y) * weight.bias
				p.Z += (joint.pos.Z + rotated.Z) * weight.bias
			}
			computedVerts[i] = p
		}

		// Resolve material
		var mat *config.Material
		if materials != nil && len(currentShader) > 0 {
			mat = materials.CreateConfigMaterial(currentShader)
		}
		if mat == nil {
			mat = config.NewConfigMaterial("md5.mesh", []string{currentShader}, config.MaterialKindSolid)
		}

		// Assemble triangles
		for _, triIdx := range currentTris {
			if triIdx[0] >= len(computedVerts) || triIdx[1] >= len(computedVerts) || triIdx[2] >= len(computedVerts) {
				continue
			}

			tri := config.NewModel3DEntryTriangle(mat)
			for k := 0; k < 3; k++ {
				vi := triIdx[k]
				v := currentVerts[vi]
				tri.Vertices[k] = config.Model3DEntryVertex{
					Pos: computedVerts[vi],
					U:   v.u,
					V:   v.v,
				}
			}
			allTriangles = append(allTriangles, tri)
		}

		currentVerts = nil
		currentWeights = nil
		currentTris = nil
		currentShader = ""
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 || strings.HasPrefix(line, "//") {
			continue
		}

		// Strip inline comments
		if commentIdx := strings.Index(line, "//"); commentIdx >= 0 {
			line = strings.TrimSpace(line[:commentIdx])
		}

		tokens := strings.Fields(line)
		if len(tokens) == 0 {
			continue
		}

		if tokens[0] == "joints" && len(tokens) > 1 && tokens[1] == "{" {
			inJoints = true
			continue
		}
		if tokens[0] == "mesh" && len(tokens) > 1 && tokens[1] == "{" {
			inMesh = true
			continue
		}
		if tokens[0] == "}" {
			if inJoints {
				inJoints = false
			} else if inMesh {
				finishMesh()
				inMesh = false
			}
			continue
		}

		if inJoints {
			// Line format: "name" parent ( px py pz ) ( qx qy qz )
			quoteStart := strings.Index(line, "\"")
			quoteEnd := strings.LastIndex(line, "\"")
			if quoteStart < 0 || quoteEnd <= quoteStart {
				continue
			}
			name := line[quoteStart+1 : quoteEnd]
			rest := strings.TrimSpace(line[quoteEnd+1:])
			subTokens := strings.Fields(rest)
			if len(subTokens) < 9 {
				continue
			}
			parent, _ := strconv.Atoi(subTokens[0])

			// Extract pos ( px py pz )
			pStart := strings.Index(rest, "(")
			pEnd := strings.Index(rest, ")")
			if pStart < 0 || pEnd <= pStart {
				continue
			}
			pFields := strings.Fields(rest[pStart+1 : pEnd])
			if len(pFields) < 3 {
				continue
			}
			px, _ := strconv.ParseFloat(pFields[0], 64)
			py, _ := strconv.ParseFloat(pFields[1], 64)
			pz, _ := strconv.ParseFloat(pFields[2], 64)

			// Extract orient ( qx qy qz )
			qRest := rest[pEnd+1:]
			qStart := strings.Index(qRest, "(")
			qEnd := strings.Index(qRest, ")")
			if qStart < 0 || qEnd <= qStart {
				continue
			}
			qFields := strings.Fields(qRest[qStart+1 : qEnd])
			if len(qFields) < 3 {
				continue
			}
			qx, _ := strconv.ParseFloat(qFields[0], 64)
			qy, _ := strconv.ParseFloat(qFields[1], 64)
			qz, _ := strconv.ParseFloat(qFields[2], 64)

			t := 1.0 - (qx*qx + qy*qy + qz*qz)
			var qw float64
			if t > 0.0 {
				qw = math.Sqrt(t)
			}

			joints = append(joints, md5Joint{
				name:   name,
				parent: parent,
				pos:    geometry.XYZ{X: px, Y: py, Z: pz},
				orient: [4]float64{qx, qy, qz, qw},
			})
			continue
		}

		if inMesh {
			switch tokens[0] {
			case "shader":
				if len(tokens) > 1 {
					currentShader = strings.Trim(tokens[1], "\"")
				}
			case "vert":
				// Format: vert index ( u v ) startWeight countWeight
				pStart := strings.Index(line, "(")
				pEnd := strings.Index(line, ")")
				if pStart >= 0 && pEnd > pStart {
					uvFields := strings.Fields(line[pStart+1 : pEnd])
					tailFields := strings.Fields(line[pEnd+1:])
					if len(uvFields) >= 2 && len(tailFields) >= 2 {
						u, _ := strconv.ParseFloat(uvFields[0], 32)
						v, _ := strconv.ParseFloat(uvFields[1], 32)
						startW, _ := strconv.Atoi(tailFields[0])
						countW, _ := strconv.Atoi(tailFields[1])

						currentVerts = append(currentVerts, md5Vert{
							u:          float32(u),
							v:          float32(v),
							weightIdx:  startW,
							weightSpan: countW,
						})
					}
				}
			case "tri":
				// Format: tri index i0 i1 i2
				if len(tokens) >= 5 {
					i0, _ := strconv.Atoi(tokens[2])
					i1, _ := strconv.Atoi(tokens[3])
					i2, _ := strconv.Atoi(tokens[4])
					currentTris = append(currentTris, [3]int{i0, i1, i2})
				}
			case "weight":
				// Format: weight index joint bias ( wx wy wz )
				pStart := strings.Index(line, "(")
				pEnd := strings.Index(line, ")")
				if pStart >= 0 && pEnd > pStart {
					headFields := strings.Fields(line[:pStart])
					posFields := strings.Fields(line[pStart+1 : pEnd])
					if len(headFields) >= 4 && len(posFields) >= 3 {
						joint, _ := strconv.Atoi(headFields[2])
						bias, _ := strconv.ParseFloat(headFields[3], 64)
						wx, _ := strconv.ParseFloat(posFields[0], 64)
						wy, _ := strconv.ParseFloat(posFields[1], 64)
						wz, _ := strconv.ParseFloat(posFields[2], 64)

						currentWeights = append(currentWeights, md5Weight{
							joint: joint,
							bias:  bias,
							pos:   geometry.XYZ{X: wx, Y: wy, Z: wz},
						})
					}
				}
			}
		}
	}

	if inMesh {
		finishMesh()
	}

	if len(allTriangles) == 0 {
		return nil, fmt.Errorf("no triangles generated from md5mesh")
	}

	cModel := config.NewModel3DEntry(1, []string{"idle"})
	cModel.Frames[0] = config.NewModel3DEntryFrame(allTriangles)
	cModel.ActionDefinitions = []string{"idle"}
	cModel.ActionIntervals = [][2]int{{0, 0}}

	cModel.DropToFloor()

	return cModel, nil
}
