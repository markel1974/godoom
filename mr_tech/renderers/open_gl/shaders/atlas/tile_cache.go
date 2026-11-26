package atlas

import (
	"unsafe"
)

// TileCache stores state used to determine whether a shadow atlas tile can be reused.
// It tracks validity against atlas generation, transform matrix, and scene hash.
type TileCache struct {
	valid     bool
	atlasGen  uint64
	matrix2   [16]float32
	sceneHash uint64
}

// NewTileCache creates a TileCache with its matrix pointer initialized to the internal matrix storage.
func NewTileCache() *TileCache {
	tc := &TileCache{}
	return tc
}

// Assign updates the tile cache entry with validity, atlas generation, transform matrix, and scene hash.
func (c *TileCache) Assign(valid bool, gen uint64, matPtr *float32, sceneHash uint64) {
	c.valid = valid
	c.atlasGen = gen
	if matPtr != nil {
		c.matrix2 = *(*[16]float32)(unsafe.Pointer(matPtr))
	} else {
		clear(c.matrix2[:])
	}
	c.sceneHash = sceneHash
}

// Reusable reports whether the cache is valid and matches the atlas generation, matrix, and scene hash.
func (c *TileCache) Reusable(atlasGen uint64, matrixPtr *float32, sceneHash uint64) bool {
	if matrixPtr == nil {
		return false
	}
	matrix := *(*[16]float32)(unsafe.Pointer(matrixPtr))
	return c.valid && c.atlasGen == atlasGen && c.sceneHash == sceneHash && c.matrix2 == matrix
}

// ReusablePeek reports whether the cache is valid and matches the atlas generation and matrix.
func (c *TileCache) ReusablePeek(atlasGen uint64, matrixPtr *float32) bool {
	if matrixPtr == nil {
		return false
	}
	matrix := *(*[16]float32)(unsafe.Pointer(matrixPtr))
	if !c.valid || c.atlasGen != atlasGen || c.matrix2 != matrix {
		return false
	}
	return true
}
