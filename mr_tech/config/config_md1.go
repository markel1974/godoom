package config

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// MD1Vertex represents a single vertex in an MD1 3D model with position and texture coordinates.
type MD1Vertex struct {
	Pos geometry.XYZ
	U   float32
	V   float32
}

// MD1Triangle represents a triangular mesh with 3 vertices and an associated material.
type MD1Triangle struct {
	Vertices [3]MD1Vertex
	Material *Material
}

// NewMD1Triangle creates a new MD1Triangle with the specified material and initializes its vertices to default values.
func NewMD1Triangle(material *Material) MD1Triangle {
	tri := MD1Triangle{
		Material: material,
	}
	return tri
}

// MD1Frame represents a collection of triangles that define a single frame in an MD1 animation sequence.
type MD1Frame struct {
	Triangles []MD1Triangle
}

// NewMD1Frame creates a new MD1Frame with the specified list of MD1Triangle structures.
func NewMD1Frame(triangles []MD1Triangle) MD1Frame {
	return MD1Frame{
		Triangles: triangles,
	}
}

// MD1 represents a structure holding animation frames, action definitions, and corresponding action intervals.
type MD1 struct {
	Frames            []MD1Frame
	ActionDefinitions []string
	ActionIntervals   [][2]int
}

// NewMD1 creates a new MD1 instance with the specified number of frames and initializes it using the provided frame names.
func NewMD1(numFrames int, frameNames []string) *MD1 {
	m := &MD1{
		Frames: make([]MD1Frame, numFrames),
	}
	m.compute(frameNames)
	return m
}

// compute processes the given frameNames, grouping frames into intervals based on their base names and populating relevant fields.
func (m *MD1) compute(frameNames []string) {
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

// getBaseName extracts the base name from a string by removing trailing numeric characters and returns the result.
func (m *MD1) getBaseName(fn string) string {
	for i := len(fn) - 1; i >= 0; i-- {
		if fn[i] < '0' || fn[i] > '9' {
			return fn[:i+1]
		}
	}
	return fn
}
