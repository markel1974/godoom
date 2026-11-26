package atlas

// TileCache stores cached projection/scene keys for a single tile in one atlas buffer.
type TileCache struct {
	valid     bool
	atlasGen  uint64
	matrix    [16]float32
	sceneHash uint64
}

// Matches reports whether the cached entry is valid for atlasGen, matrix, and sceneHash.
func (c *TileCache) Matches(atlasGen uint64, matrix *[16]float32, sceneHash uint64) bool {
	return matrix != nil && c.valid && c.atlasGen == atlasGen && c.sceneHash == sceneHash && c.matrix == *matrix
}
