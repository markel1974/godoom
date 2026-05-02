package lumps

// model_agnostic.go (nel package lumps)

type RawModel struct {
	FirstFace int32
	NumFaces  int32
}

type RawFace struct {
	FirstEdge int32
	NumEdges  uint16
	TexName   string // Semplificato: il nome esatto della texture pronto all'uso
}

type RawVertex struct {
	X, Y, Z float32
}
