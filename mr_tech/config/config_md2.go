package config

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// MD2Vertex represents a vertex in an MD2 model with a position and texture coordinates (U, V).
type MD2Vertex struct {
	Pos geometry.XYZ
	U   float32
	V   float32
}

// MD2Frame represents a single frame in an MD2 animation, consisting of an array of triangles made up of MD2Vertex data.
type MD2Frame struct {
	Triangles [][3]MD2Vertex
}

// NewMD2Frame creates a new MD2Frame with the specified number of triangles, initializing the Triangles slice accordingly.
func NewMD2Frame(numTris int) MD2Frame {
	return MD2Frame{Triangles: make([][3]MD2Vertex, numTris)}
}

// MD2 represents a 3D model composed of animation frames, associated actions, and frame intervals.
type MD2 struct {
	Frames            []MD2Frame
	ActionDefinitions []string
	ActionIntervals   [][2]int
}

// NewMD2 creates a new MD2 object with the specified number of frames and frame names, initializing its data structures.
func NewMD2(numFrames int, frameNames []string) *MD2 {
	m := &MD2{
		Frames: make([]MD2Frame, numFrames),
	}
	m.compute(frameNames)
	return m
}

// getBaseName extracts the base name from the given string by removing trailing numeric characters.
func (m *MD2) getBaseName(fn string) string {
	for i := len(fn) - 1; i >= 0; i-- {
		if fn[i] < '0' || fn[i] > '9' {
			return fn[:i+1]
		}
	}
	return fn
}

// compute processes frame names, grouping them into intervals and associating them with corresponding action names.
func (m *MD2) compute(frameNames []string) {
	if len(frameNames) == 0 {
		return
	}
	currentBase := m.getBaseName(frameNames[0])
	startIdx := 0
	for i := 1; i <= len(frameNames); i++ {
		var base string
		if i < len(frameNames) {
			base = m.getBaseName(frameNames[i])
		}
		if i == len(frameNames) || base != currentBase {
			if startIdx < 0 || startIdx > len(m.Frames) {
				fmt.Println("startIdx out of range")
				continue
			}
			endIdx := i - 1
			if endIdx < 0 || endIdx >= len(m.Frames) {
				fmt.Println("endIdx out of range")
				continue
			}
			m.ActionIntervals = append(m.ActionIntervals, [2]int{startIdx, i - 1})
			m.ActionDefinitions = append(m.ActionDefinitions, currentBase)
			if i < len(frameNames) {
				currentBase = base
				startIdx = i
			}
		}
	}
}
