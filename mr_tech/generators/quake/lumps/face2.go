package lumps

// Face2 represents a face in the Quake 2 BSP map format.
// PlaneID is the plane index in the BSP plane array associated with this face.
// Side specifies whether the face is oriented in the same or opposite direction to the plane.
// FirstEdge is the starting index in the surface edge array for this face's edges.
// NumEdges indicates the total number of edges defining this face.
// TexInfo is the index into the texture information array for texture details of the face.
// LightTypes contains light style indices for the face's dynamic lighting data.
// Lightmap is the offset in the lightmap data where this face's lightmap starts.
type Face2 struct {
	PlaneID    uint16
	Side       uint16
	FirstEdge  int32
	NumEdges   uint16
	TexInfo    uint16
	LightTypes [4]uint8
	Lightmap   int32
}
