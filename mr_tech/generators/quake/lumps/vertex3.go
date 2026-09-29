package lumps

// Vertex3 represents a 3D vertex with position, texture coordinates, light map coordinates, normal, and color attributes.
type Vertex3 struct {
	Position  [3]float32
	TexCoord  [2]float32
	LMapCoord [2]float32
	Normal    [3]float32
	Color     [4]uint8
}
