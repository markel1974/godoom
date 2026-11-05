package md5

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/geometry"
)

// MeshVertex represents a vertex definition in the MD5 mesh.
type MeshVertex struct {
	U          float32
	V          float32
	WeightIdx  int
	WeightSpan int
}

// MeshWeight represents a vertex skinning weight attached to a specific Joint.
type MeshWeight struct {
	JointId int
	Bias    float64
	Pos     geometry.XYZ
}

// SubMesh holds the parsed geometry and weight list of an individual sub-mesh.
type SubMesh struct {
	Shader   string
	Vertices []MeshVertex
	Tris     [][3]int
	Weights  []MeshWeight
}

// Mesh represents a parsed MD5 mesh structure containing joints and sub-mesh data.
type Mesh struct {
	Joints    []*Joint
	SubMeshes []SubMesh
}

// NewMesh creates and returns a new instance of Mesh with uninitialized joints and sub-mesh data.
func NewMesh() *Mesh {
	return &Mesh{}
}

// Parse reads an Mesh file from the provided reader and populates the mesh with parsed joints and sub-mesh data.
func (mesh *Mesh) Parse(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	inJoints := false
	inMesh := false

	var currentSubMesh SubMesh

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 || strings.HasPrefix(line, "//") {
			continue
		}

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
			currentSubMesh = SubMesh{}
			continue
		}
		if tokens[0] == "}" {
			if inJoints {
				inJoints = false
			} else if inMesh {
				if len(currentSubMesh.Vertices) > 0 && len(currentSubMesh.Tris) > 0 {
					mesh.SubMeshes = append(mesh.SubMeshes, currentSubMesh)
				}
				inMesh = false
			}
			continue
		}

		if inJoints {
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

			// In Doom 3 MD5 format, qw is strictly negative
			t := 1.0 - (qx*qx + qy*qy + qz*qz)
			var qw float64
			if t > 0.0 {
				qw = -math.Sqrt(t)
			}

			mesh.Joints = append(mesh.Joints, NewJoint(name, parent, geometry.XYZ{X: px, Y: py, Z: pz}, [4]float64{qx, qy, qz, qw}))
			continue
		}

		if inMesh {
			switch tokens[0] {
			case "shader":
				if len(tokens) > 1 {
					currentSubMesh.Shader = strings.Trim(tokens[1], "\"")
				}
			case "vert":
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

						currentSubMesh.Vertices = append(currentSubMesh.Vertices, MeshVertex{
							U:          float32(u),
							V:          float32(v),
							WeightIdx:  startW,
							WeightSpan: countW,
						})
					}
				}
			case "tri":
				if len(tokens) >= 5 {
					i0, _ := strconv.Atoi(tokens[2])
					i1, _ := strconv.Atoi(tokens[3])
					i2, _ := strconv.Atoi(tokens[4])
					currentSubMesh.Tris = append(currentSubMesh.Tris, [3]int{i0, i1, i2})
				}
			case "weight":
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

						currentSubMesh.Weights = append(currentSubMesh.Weights, MeshWeight{
							JointId: joint,
							Bias:    bias,
							Pos:     geometry.XYZ{X: wx, Y: wy, Z: wz},
						})
					}
				}
			}
		}
	}

	if inMesh && len(currentSubMesh.Vertices) > 0 && len(currentSubMesh.Tris) > 0 {
		mesh.SubMeshes = append(mesh.SubMeshes, currentSubMesh)
	}

	if len(mesh.SubMeshes) == 0 {
		return fmt.Errorf("no valid sub-meshes parsed from md5mesh")
	}

	return nil
}
