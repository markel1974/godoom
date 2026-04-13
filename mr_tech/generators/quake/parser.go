package quake

import (
	"fmt"
	"math"
	"strconv"

	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// Builder is a utility type that provides methods to construct and manage texture resources and scaling factors.
type Builder struct {
	texManager *Textures
}

// NewBuilder creates and returns a new Builder instance with the specified scale. Defaults to 1.0 if scale is non-positive.
func NewBuilder() *Builder {
	return &Builder{
		texManager: NewTextures(),
	}
}

// Setup initializes the Builder by loading resources from the specified PAK file and preparing the level configuration.
func (p *Builder) Setup(pakPath string, level int) (*config.ConfigRoot, error) {
	bpsPath := "maps" + lumps.PakSeparator + "e1m1.bsp"
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
	leaves, _ := lumps.NewLeaves(rs, infos[lumps.LumpLeaves])
	entities, _ := lumps.NewEntities(rs, infos[lumps.LumpEntities])
	marks, _ := lumps.NewMarks(rs, infos[lumps.LumpMarkSurfaces])

	for _, mt := range mipTextures {
		if mt != nil && mt.Name != "" {
			if err = p.texManager.RegisterPixels(mt.Name, int(mt.Width), int(mt.Height), mt.Pixels[0], palette); err != nil {
				fmt.Printf("Warning: texture %s error: %s\n", mt.Name, err.Error())
			}
		}
	}

	// Inizializzazione ConfigRoot con il riferimento al gestore texture popolato
	player := config.NewConfigPlayer(geometry.XYZ{}, 0, 8, 4, 20)
	root := config.NewConfigRoot(nil, player, nil, 1.0, true, p.texManager)
	root.Full3d = true

	// 3. Parsing delle Entità (Luci, Player, Monsters)
	for _, ent := range entities {
		classname := ent.Properties["classname"]
		var pos geometry.XYZ
		if origin, ok := ent.Properties["origin"]; ok {
			var x, y, z float64
			_, _ = fmt.Sscanf(origin, "%f %f %f", &x, &y, &z)
			// Conversione coordinate: Quake Z-up -> Engine Y-up
			pos = geometry.XYZ{X: x, Y: z, Z: -y}
		}
		switch {
		case classname == "info_player_start":
			player.Position = pos
			if angle, ok := ent.Properties["angle"]; ok {
				if val, err := strconv.ParseFloat(angle, 64); err == nil {
					player.Angle = val * (math.Pi / 180.0)
				}
			}

		case classname == "light":
			intensity := 300.0
			falloff := 32.0
			if l, ok := ent.Properties["light"]; ok {
				if val, err := strconv.ParseFloat(l, 64); err == nil {
					intensity = val
					falloff = intensity / 10.0
				}
			}
			root.Lights = append(root.Lights, config.NewConfigLight(pos, intensity, config.LightKindSpot, falloff))
		}
	}

	// 4. Conversione Geometria: BSP Leaves -> ConfigVolume
	for leafIdx, leaf := range leaves {
		// Saltiamo la leaf 0 (solitamente spazio solido esterno)
		if leafIdx == 0 {
			continue
		}

		volId := fmt.Sprintf("leaf_%d", leafIdx)
		volume := config.NewConfigVolume(volId, "quake_bsp")

		for i := uint16(0); i < leaf.NumFaces; i++ {
			faceIdx := marks.Surfaces[leaf.FirstFace+i]
			bspFace := faces[faceIdx]
			texInfo := texInfos[bspFace.TexInfo]

			// Risoluzione ID Materiale
			texName := "default"
			isSky := (texInfo.Flags & 4) != 0
			if texInfo.MipTex < uint32(len(mipTextures)) && mipTextures[texInfo.MipTex] != nil {
				texName = mipTextures[texInfo.MipTex].Name
			}

			// Ricostruzione Poligono 3D dai SurfEdges
			var points []geometry.XYZ
			for j := uint16(0); j < bspFace.NumEdges; j++ {
				surfEdgeIdx := surfEdges[bspFace.FirstEdge+int32(j)]
				var v *lumps.Vertex
				if surfEdgeIdx >= 0 {
					v = vertexes[edges[surfEdgeIdx].Vertex0]
				} else {
					v = vertexes[edges[-surfEdgeIdx].Vertex1]
				}
				points = append(points, geometry.XYZ{X: float64(v.X), Y: float64(v.Z), Z: float64(-v.Y)})
			}

			// Creazione Animazione (Materiale) con l'ID texture registrato
			animKind := config.AnimationKindLoop
			if isSky {
				animKind = config.AnimationKindSky
			}
			material := config.NewConfigAnimation([]string{texName}, animKind, 1.0, 1.0)
			// Inserimento della faccia nel volume con ID materiale coerente
			volume.Faces = append(volume.Faces, config.NewConfigFace(points, material, texName))
		}

		if len(volume.Faces) > 0 {
			root.Volumes = append(root.Volumes, volume)
		}
	}

	return root, nil
}
