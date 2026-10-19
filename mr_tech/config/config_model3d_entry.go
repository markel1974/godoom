package config

import (
	"fmt"
	"strconv"

	"github.com/markel1974/godoom/mr_tech/geometry"
)

// Model3DEntryVertex represents a single vertex in an Model3DEntry 3D model with position and texture coordinates.
type Model3DEntryVertex struct {
	Pos geometry.XYZ
	U   float32
	V   float32
}

// Model3DEntryTriangle represents a triangular mesh with 3 vertices and an associated material.
type Model3DEntryTriangle struct {
	Vertices [3]Model3DEntryVertex
	Material *Material
}

// NewModel3DEntryTriangle creates a new Model3DEntryTriangle with the specified material and initializes its vertices to default values.
func NewModel3DEntryTriangle(material *Material) Model3DEntryTriangle {
	tri := Model3DEntryTriangle{
		Material: material,
	}
	return tri
}

// Model3DTag represents a coordinate attachment tag with spatial origin and 3x3 orientation axes.
type Model3DTag struct {
	Origin geometry.XYZ
	Axis   [3]geometry.XYZ
}

// NewModel3DTag creates a new Model3DTag with the specified origin and orientation axis.
func NewModel3DTag(origin geometry.XYZ, axis [3]geometry.XYZ) Model3DTag {
	return Model3DTag{
		Origin: origin,
		Axis:   axis,
	}
}

// Model3DEntryFrame represents a collection of triangles that define a single frame in an Model3DEntry animation sequence.
type Model3DEntryFrame struct {
	Triangles []Model3DEntryTriangle
	Tags      map[string]Model3DTag
}

// NewModel3DEntryFrame creates a new Model3DEntryFrame with the specified list of Model3DEntryTriangle structures.
func NewModel3DEntryFrame(triangles []Model3DEntryTriangle) Model3DEntryFrame {
	return Model3DEntryFrame{
		Triangles: triangles,
		Tags:      make(map[string]Model3DTag),
	}
}

// Model3DEntry represents a structure holding animation frames, action definitions, and corresponding action intervals.
type Model3DEntry struct {
	Frames            []Model3DEntryFrame
	ActionClamp       []string
	ActionDefinitions []string
	ActionIntervals   [][2]int
}

// NewModel3DEntry creates a new Model3DEntry instance with the specified number of frames and initializes it using the provided frame names.
func NewModel3DEntry(numFrames int, frameNames []string) *Model3DEntry {
	m := &Model3DEntry{
		Frames: make([]Model3DEntryFrame, numFrames),
	}
	for i := 0; i < numFrames; i++ {
		m.Frames[i].Tags = make(map[string]Model3DTag)
	}
	m.ActionClamp = []string{"death", "die"}
	m.compute(frameNames)
	return m
}

// compute processes the given frameNames, grouping frames into intervals based on their base names and populating relevant fields.
func (m *Model3DEntry) compute(frameNames []string) {
	if len(frameNames) == 0 {
		return
	}
	currentBase, counter := m.getBaseName(frameNames[0])
	startIdx := 0
	for i := 1; i <= len(frameNames); i++ {
		var base string
		update := false
		if i < len(frameNames) {
			var nc int
			base, nc = m.getBaseName(frameNames[i])
			if base != currentBase {
				update = true
			} else if nc != counter+1 {
				update = true
			}
			counter = nc
		}
		if i == len(frameNames) {
			update = true
		}
		if update {
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
			name := currentBase + strconv.Itoa(len(m.ActionDefinitions))
			m.ActionDefinitions = append(m.ActionDefinitions, name)
			if i < len(frameNames) {
				currentBase = base
				startIdx = i
			}
		}
	}
}

// getBaseName extracts the base name from a string by removing trailing numeric characters and returns the result.
func (m *Model3DEntry) getBaseName(fn string) (string, int) {
	pos := -1
	for x := 0; x < len(fn); x++ {
		t := fn[x]
		if t >= '0' && t <= '9' {
			pos = x
			break
		}
	}
	if pos < 0 {
		return fn, 0
	}
	a := fn[:pos]
	num := fn[pos:]
	val, err := strconv.Atoi(num)
	if err != nil {
		fmt.Println("Error converting string to int:", err)
		return a, 0
	}
	return a, val
}
