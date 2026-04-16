package quake

import (
	"fmt"
	"math"
	"strconv"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

//MODEL IMPORTAL
//model is Z up
//model is CCW

// Builder is responsible for constructing and managing graphical elements with the aid of a Textures manager.
type Builder struct {
	texManager *Textures
}

// NewBuilder creates and initializes a new Builder instance with a default Textures manager.
func NewBuilder() *Builder {
	return &Builder{
		texManager: NewTextures(),
	}
}

// Setup initializes the Builder by loading level data and assets from the specified PAK file and level index.
func (p *Builder) Setup(pakPath string, level int) (*config.Root, error) {
	bpsPath := "maps" + lumps.PakSeparator + "e1m" + strconv.Itoa(level) + ".bsp"
	palPath := "gfx" + lumps.PakSeparator + "palette.lmp"
	pk := lumps.NewPak()
	if err := pk.Setup(pakPath); err != nil {
		return nil, err
	}
	palReader, err := pk.Open(palPath)
	if err != nil {
		return nil, err
	}
	palette, err := lumps.NewPalette(palReader)
	if err != nil {
		return nil, err
	}
	rs, err := pk.Open(bpsPath)
	if err != nil {
		return nil, err
	}
	// Header
	infos, err := lumps.NewLumpInfos(rs)
	if err != nil {
		return nil, err
	}
	// Geometry
	vertexes, _ := lumps.NewVertexes(rs, infos[lumps.LumpVertexes])
	edges, _ := lumps.NewEdges(rs, infos[lumps.LumpEdges])
	surfEdges, _ := lumps.NewSurfEdges(rs, infos[lumps.LumpSurfEdges])
	faces, _ := lumps.NewFace(rs, infos[lumps.LumpFaces])
	texInfos, _ := lumps.NewTexInfos(rs, infos[lumps.LumpTexInfo])
	mipTextures, _ := lumps.NewMipTextures(rs, infos[lumps.LumpTextures])
	//leaves, _ := lumps.NewLeaves(rs, infos[lumps.LumpLeaves])
	entities, _ := lumps.NewEntities(rs, infos[lumps.LumpEntities])
	//marks, _ := lumps.NewMarks(rs, infos[lumps.LumpMarkSurfaces])
	bspModels, _ := lumps.NewModels(rs, infos[lumps.LumpModels])

	for _, mt := range mipTextures {
		if mt != nil && mt.Name != "" {
			if err = p.texManager.RegisterPixels(mt.Name, int(mt.Width), int(mt.Height), mt.Pixels[0], palette); err != nil {
				fmt.Printf("Warning: texture %s error: %s\n", mt.Name, err.Error())
			}
		}
	}

	var playerAngle float64
	var playerPos geometry.XYZ
	root := config.NewConfigRoot(nil, nil, nil, 1.0, p.texManager)
	root.Full3d = true

	// 3. Parsing delle Entità (Luci, Player, Monsters)
	for _, ent := range entities {
		classname := ent.Properties["classname"]
		var pos geometry.XYZ
		if origin, ok := ent.Properties["origin"]; ok {
			var x, y, z float64
			_, _ = fmt.Sscanf(origin, "%f %f %f", &x, &y, &z)
			pos = CreateXYZ(x, y, z)
		}
		switch {
		case classname == "info_player_start":
			playerPos = pos
			if angle, ok := ent.Properties["angle"]; ok {
				if val, err := strconv.ParseFloat(angle, 64); err == nil {
					playerAngle = val * (math.Pi / 180.0)
				}
			}

		case classname == "light":
			intensity := 200.0
			falloff := 128.0
			if l, ok := ent.Properties["light"]; ok {
				if val, err := strconv.ParseFloat(l, 64); err == nil {
					intensity = val
					falloff = intensity / 10.0
				}
			}
			root.Lights = append(root.Lights, config.NewConfigLight(pos, intensity, config.LightKindAmbient, falloff))
		}
	}

	// 4. Conversione Geometria Statica: BSP Faces -> Volume
	// Creiamo un singolo volume globale, senza duplicazioni.
	volume := config.NewConfigVolume("quake_world", "quake_bsp")

	worldModel := bspModels[0]

	// Iteriamo ESCLUSIVAMENTE sulle facce che appartengono al mondo
	for i := int32(0); i < worldModel.NumFaces; i++ {
		faceIdx := worldModel.FirstFace + i
		bspFace := faces[faceIdx]
		texInfo := texInfos[bspFace.TexInfo]
		texName := "default"
		isSky := (texInfo.Flags & 4) != 0
		if texInfo.MipTex < uint32(len(mipTextures)) && mipTextures[texInfo.MipTex] != nil {
			texName = mipTextures[texInfo.MipTex].Name
		}
		var points []geometry.XYZ
		for j := uint16(0); j < bspFace.NumEdges; j++ {
			surfEdgeIdx := surfEdges[bspFace.FirstEdge+int32(j)]
			var v *lumps.Vertex
			if surfEdgeIdx >= 0 {
				v = vertexes[edges[surfEdgeIdx].Vertex0]
			} else {
				v = vertexes[edges[-surfEdgeIdx].Vertex1]
			}
			pos := CreateXYZ(float64(v.X), float64(v.Y), float64(v.Z))
			points = append(points, pos)
		}
		animKind := config.AnimationKindLoop
		if isSky {
			animKind = config.AnimationKindSky
		}
		material := config.NewConfigAnimation([]string{texName}, animKind, 1.0, 1.0)
		triangles := TriangulateConvex3d(points)
		for _, tri := range triangles {
			volume.Faces = append(volume.Faces, config.NewConfigFace(tri, material, texName))
		}
	}
	if len(volume.Faces) > 0 {
		root.Volumes = append(root.Volumes, volume)
	}
	root.Player = config.NewConfigPlayer(playerPos, playerAngle, 40, 4, 80)
	root.Player.Speed = 800
	return root, nil
}

// CreateXYZ constructs a geometry.XYZ object using the provided x, y, and z coordinate values.
func CreateXYZ(x, y, z float64) geometry.XYZ {
	// Conversione coordinate: Quake Z-up -> Engine Z-up
	//pos := geometry.XYZ{X: x, Y: z, Z: -y}
	pos := geometry.XYZ{X: x, Y: y, Z: z}
	return pos
}

// TriangulateConvex3d splits a convex 3D polygon into triangles, using the first vertex as a common anchor point.
// pts is a slice of 3D points representing the convex polygon. Returns a slice of triangle slices or nil if invalid.
func TriangulateConvex3d(pts []geometry.XYZ) [][]geometry.XYZ {
	pLen := len(pts)
	if pLen < 3 {
		return nil // Poligono degenere
	}
	if pLen == 3 {
		return [][]geometry.XYZ{{pts[0], pts[1], pts[2]}}
	}
	output := make([][]geometry.XYZ, 0, pLen-2)
	// Triangle Fan ancorato a pts[0]
	for i := 1; i < pLen-1; i++ {
		output = append(output, []geometry.XYZ{pts[0], pts[i], pts[i+1]})
	}
	return output
}

// TriangulateConvex3dInverted generates inverted triangle definitions for a 3D convex polygon.
// Accepts a slice of geometry.XYZ points and returns a slice of triangle slices.
// Returns nil if fewer than 3 points are provided.
func TriangulateConvex3dInverted(pts []geometry.XYZ) [][]geometry.XYZ {
	pLen := len(pts)
	if pLen < 3 {
		return nil
	}
	if pLen == 3 {
		// INVERTITO: da (0, 1, 2) a (0, 2, 1)
		return [][]geometry.XYZ{{pts[0], pts[2], pts[1]}}
	}

	output := make([][]geometry.XYZ, 0, pLen-2)
	for i := 1; i < pLen-1; i++ {
		// INVERTITO: pts[i+1] viene PRIMA di pts[i]
		output = append(output, []geometry.XYZ{pts[0], pts[i+1], pts[i]})
	}
	return output
}
