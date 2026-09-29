package lumps

// Model2 represents a 3D model's bounding box, spatial properties, and associated geometric data in a scene graph.
type Model2 struct {
	Mins      [3]float32
	Maxs      [3]float32
	Origin    [3]float32
	HeadNode  int32
	FirstFace int32
	NumFaces  int32
}
