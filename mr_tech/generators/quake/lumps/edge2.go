package lumps

// Edge2 represents an edge in a Quake 2 BSP file, defined by two vertex indices V1 and V2.
type Edge2 struct {
	V1, V2 uint16
}
