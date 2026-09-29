package lumps

// LumpEntities2 represents the lump ID for entities.
// LumpPlanes2 represents the lump ID for planes.
// LumpVertexes2 represents the lump ID for vertex data.
// LumpVisibility2 represents the lump ID for visibility information.
// LumpNodes2 represents the lump ID for BSP tree nodes.
// LumpTexInfo2 represents the lump ID for texture information.
// LumpFaces2 represents the lump ID for face definitions.
// LumpLighting2 represents the lump ID for lighting data.
// LumpLeaves2 represents the lump ID for BSP tree leaves.
// LumpLeafFaces2 represents the lump ID that maps leaves to faces.
// LumpLeafBrushes2 represents the lump ID that maps leaves to brushes.
// LumpEdges2 represents the lump ID for edge definitions.
// LumpSurfEdges2 represents the lump ID for surface edge mappings.
// LumpModels2 represents the lump ID for models within the BSP file.
// LumpBrushes2 represents the lump ID for brush definitions.
// LumpBrushSides2 represents the lump ID for brush side definitions.
// LumpPop2 represents the lump ID for the PVS clusters offset table.
// LumpAreas2 represents the lump ID for area definitions.
// LumpAreaPortals2 represents the lump ID for area portal definitions.
// NumLumps2 defines the total number of lumps.
const (
	LumpEntities2    = 0
	LumpPlanes2      = 1
	LumpVertexes2    = 2
	LumpVisibility2  = 3
	LumpNodes2       = 4
	LumpTexInfo2     = 5
	LumpFaces2       = 6
	LumpLighting2    = 7
	LumpLeaves2      = 8
	LumpLeafFaces2   = 9
	LumpLeafBrushes2 = 10
	LumpEdges2       = 11
	LumpSurfEdges2   = 12
	LumpModels2      = 13
	LumpBrushes2     = 14
	LumpBrushSides2  = 15
	LumpPop2         = 16
	LumpAreas2       = 17
	LumpAreaPortals2 = 18
	NumLumps2        = 19
)

// Headers2 represents the structure of a file header containing a magic identifier, version, and lump metadata.
type Headers2 struct {
	Magic   [4]byte
	Version int32
	Lumps   [NumLumps2]struct {
		Offset int32
		Length int32
	}
}
