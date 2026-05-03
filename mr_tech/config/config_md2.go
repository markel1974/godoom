package config

import "github.com/markel1974/godoom/mr_tech/model/geometry"

// MD2Vertex represents a 3D vertex with position coordinates and UV texture mapping coordinates.
type MD2Vertex struct {
	Pos geometry.XYZ
	U   float32
	V   float32
}

// MD2Frame represents a 3D frame composed of a collection of triangles, each defined by three MD2Vertex points.
type MD2Frame struct {
	Triangles [][3]MD2Vertex
}

func NewMD2Frame(numTris int) MD2Frame {
	return MD2Frame{Triangles: make([][3]MD2Vertex, numTris)}
}

// MD2 represents a 3D model containing multiple frames, where each frame is a collection of triangular geometric data.
type MD2 struct {
	Frames []MD2Frame
	Names  []string
}

func NewMD2(numFrames int, frameNames []string) *MD2 {
	return &MD2{
		Frames: make([]MD2Frame, numFrames),
		Names:  frameNames,
	}
}

// ExtractAnimations TODO
func ExtractAnimations(frameNames []string) map[string][2]int {
	getBaseName := func(fn string) string {
		for i := len(fn) - 1; i >= 0; i-- {
			if fn[i] < '0' || fn[i] > '9' {
				return fn[:i+1]
			}
		}
		return fn
	}

	animMap := make(map[string][2]int)
	if len(frameNames) == 0 {
		return animMap
	}

	currentBase := getBaseName(frameNames[0])
	startIdx := 0

	for i := 1; i <= len(frameNames); i++ {
		var base string
		if i < len(frameNames) {
			base = getBaseName(frameNames[i])
		}
		// Se il nome base cambia (o siamo all'ultimo frame), chiudiamo l'intervallo
		if i == len(frameNames) || base != currentBase {
			animMap[currentBase] = [2]int{startIdx, i - 1}
			if i < len(frameNames) {
				currentBase = base
				startIdx = i
			}
		}
	}
	return animMap
}
