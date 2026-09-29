package lumps

// Face3 represents a geometric face in a 3D model with associated metadata and attributes.
type Face3 struct {
	TextureID   int32
	Effect      int32
	Type        int32 // 1=Polygon, 2=Patch, 3=Mesh, 4=Billboard
	VertexStart int32
	NumVertexes int32
	MeshStart   int32
	NumMesh     int32
	LightmapID  int32
	LMapCorner  [2]int32
	LMapSize    [2]int32
	LMapOrigin  [3]float32
	LMapVecs    [2][3]float32
	Normal      [3]float32
	PatchSize   [2]int32
}
