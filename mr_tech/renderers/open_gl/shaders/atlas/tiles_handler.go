package atlas

// TilesHandler manages the fixed set of shadow tile descriptors used by a shadow atlas.
type TilesHandler struct {
	tiles [MaxShadowTiles]*TileShadow
}

// NewTilesHandler creates a tiles handler with all shadow tile slots initialized.
func NewTilesHandler() *TilesHandler {
	t := &TilesHandler{}
	for x := range t.tiles {
		t.tiles[x] = NewTileShadow()
	}
	return t
}

// Get returns the shadow tile at index x.
func (t *TilesHandler) Get(x int) *TileShadow {
	return t.tiles[x]
}

// DisableAllTiles marks every shadow tile inactive and clears its reuse state.
func (w *TilesHandler) DisableAllTiles() {
	for t := range MaxShadowTiles {
		w.tiles[t].SetActive(false)
		w.tiles[t].SetReuse(false)
	}
}
