package lumps

// Model3 represents a 3D model in a Quake 3 BSP file, including bounds and references to associated geometry data.
type Model3 struct {
	Mins       [3]float32
	Maxs       [3]float32
	FirstFace  int32
	NumFaces   int32
	FirstBrush int32
	NumBrushes int32
}
