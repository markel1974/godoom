package gpu

type faceStateKey struct {
	blendMode     int
	cullMode      int
	hasCutout     bool
	depthWrite    bool
	polygonOffset bool
	sortOrder     int
}

// ResidentSubMesh describes a contiguous index slice inside a ResidentMesh grouped by render state.
type ResidentSubMesh struct {
	BlendMode     int
	CullMode      int
	HasCutout     bool
	DepthWrite    bool
	PolygonOffset bool
	SortOrder     int
	CentroidX     float32
	CentroidY     float32
	CentroidZ     float32
	FirstIndex    int32
	IndexCount    int32
}

// FinalizeCentroid stores the averaged world-space centroid on the submesh when count is positive.
func (sub *ResidentSubMesh) FinalizeCentroid(sumX, sumY, sumZ float64, count int) {
	if sub != nil && count > 0 {
		inv := 1.0 / float64(count)
		sub.CentroidX = float32(sumX * inv)
		sub.CentroidY = float32(sumY * inv)
		sub.CentroidZ = float32(sumZ * inv)
	}
}
