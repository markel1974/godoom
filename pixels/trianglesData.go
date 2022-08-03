package pixels

import (
	"fmt"
)

type TriangleData struct {
	Position  Vec
	Color     RGBA
	Picture   Vec
	Intensity float64
	ClipRect  Rect
	IsClipped bool
}

// zeroValueTriangleData is the default value of a TriangleData element
var zeroValueTriangleData = TriangleData{Color: RGBA{1, 1, 1, 1}}

// TrianglesData specifies a list of ITriangles vertices with three common properties:
// ITrianglesPosition, ITrianglesColor and ITrianglesPicture.
type TrianglesData []TriangleData

// MakeTrianglesData creates TrianglesData of length len initialized with default property values.
//
// Prefer this function to make(TrianglesData, len), because make zeros them, while this function
// does the correct initialization.
func MakeTrianglesData(len int) *TrianglesData {
	td := make(TrianglesData, len)
	for i := 0; i < len; i++ {
		td[i] = zeroValueTriangleData
	}
	return &td
}

// Len returns the number of vertices in TrianglesData.
func (td *TrianglesData) Len() int {
	return len(*td)
}

// SetLen resizes TrianglesData to len, while keeping the original content.
//
// If len is greater than TrianglesData's current length, the new data is filled with default
// values ((0, 0), white, (0, 0), 0).
func (td *TrianglesData) SetLen(len int) {
	if len > td.Len() {
		needAppend := len - td.Len()
		for i := 0; i < needAppend; i++ {
			*td = append(*td, zeroValueTriangleData)
		}
	}
	if len < td.Len() {
		*td = (*td)[:len]
	}
}

// Slice returns a sub-ITriangles of this TrianglesData.
func (td *TrianglesData) Slice(i, j int) ITriangles {
	s := (*td)[i:j]
	return &s
}

func (td *TrianglesData) updateData(t ITriangles) {
	// fast path optimization
	if t, ok := t.(*TrianglesData); ok {
		copy(*td, *t)
		return
	}

	// slow path manual copy
	if t, ok := t.(ITrianglesPosition); ok {
		for i := range *td {
			(*td)[i].Position = t.Position(i)
		}
	}
	if t, ok := t.(ITrianglesColor); ok {
		for i := range *td {
			(*td)[i].Color = t.Color(i)
		}
	}
	if t, ok := t.(ITrianglesPicture); ok {
		for i := range *td {
			(*td)[i].Picture, (*td)[i].Intensity = t.Picture(i)
		}
	}
	if t, ok := t.(ITrianglesClipped); ok {
		for i := range *td {
			(*td)[i].ClipRect, (*td)[i].IsClipped = t.ClipRect(i)
		}
	}
}

// Update copies vertex properties from the supplied ITriangles into this TrianglesData.
//
// ITrianglesPosition, ITrianglesColor and TrianglesTexture are supported.
func (td *TrianglesData) Update(t ITriangles) {
	if td.Len() != t.Len() {
		panic(fmt.Errorf("(%T).Update: invalid triangles length", td))
	}
	td.updateData(t)
}

// Copy returns an exact independent copy of this TrianglesData.
func (td *TrianglesData) Copy() ITriangles {
	copyTd := MakeTrianglesData(td.Len())
	copyTd.Update(td)
	return copyTd
}

// Position returns the position property of i-th vertex.
func (td *TrianglesData) Position(i int) Vec {
	return (*td)[i].Position
}

// Color returns the color property of i-th vertex.
func (td *TrianglesData) Color(i int) RGBA {
	return (*td)[i].Color
}

// IPicture returns the picture property of i-th vertex.
func (td *TrianglesData) Picture(i int) (pic Vec, intensity float64) {
	return (*td)[i].Picture, (*td)[i].Intensity
}

// ClipRect returns the clipping rectangle property of the i-th vertex.
func (td *TrianglesData) ClipRect(i int) (rect Rect, has bool) {
	return (*td)[i].ClipRect, (*td)[i].IsClipped
}
