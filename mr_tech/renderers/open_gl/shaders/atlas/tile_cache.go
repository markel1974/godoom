package atlas

// TileCache stores state used to determine whether a shadow atlas tile can be reused.
// It tracks validity against atlas generation, transform matrix, and scene hash.
type TileCache struct {
	valid     bool
	atlasGen  uint64
	matrix2   [16]float32
	matrixPtr *float32
	sceneHash uint64
}

// NewTileCache creates a TileCache with its matrix pointer initialized to the internal matrix storage.
func NewTileCache() *TileCache {
	tc := &TileCache{}
	tc.matrixPtr = &tc.matrix2[0]
	return tc
}

// Assign updates the tile cache entry with validity, atlas generation, transform matrix, and scene hash.
func (c *TileCache) Assign(valid bool, gen uint64, matrix [16]float32, sceneHash uint64) {
	c.valid = valid
	c.atlasGen = gen
	copy(c.matrix2[:], matrix[:])
	c.sceneHash = sceneHash
}

// Matches reports whether the cache is valid and matches the atlas generation, matrix, and scene hash.
func (c *TileCache) Matches(atlasGen uint64, matrix *[16]float32, sceneHash uint64) bool {
	return matrix != nil && c.valid && c.atlasGen == atlasGen && c.sceneHash == sceneHash && c.matrix2 == *matrix
}
