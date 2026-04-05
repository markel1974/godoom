package quake

import (
	"encoding/binary"
	"fmt"
	"os"
	"strconv"

	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// Parser gestisce il caricamento e la conversione di un file Quake BSP in ConfigRoot.
type Parser struct{}

// NewParser crea una nuova istanza del parser Quake.
func NewParser() *Parser {
	return &Parser{}
}

// Parse esegue il caricamento dei lump e la costruzione della scena 3D.
func (p *Parser) Parse(filename string) (*config.ConfigRoot, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// 1. Caricamento Metadati Lump
	infos, err := lumps.NewLumpInfos(f)
	if err != nil {
		return nil, err
	}

	// 2. Deserializzazione Lumps Geometrici e Texture
	vertexes, _ := lumps.NewVertexes(f, infos[lumps.LumpVertexes])       //
	edges, _ := lumps.NewEdges(f, infos[lumps.LumpEdges])                //
	surfEdges, _ := lumps.NewSurfEdges(f, infos[lumps.LumpSurfEdges])    //
	faces, _ := lumps.NewFace(f, infos[lumps.LumpFaces])                 //
	texInfos, _ := lumps.NewTexInfos(f, infos[lumps.LumpTexInfo])        //
	mipTextures, _ := lumps.NewMipTextures(f, infos[lumps.LumpTextures]) //
	leaves, _ := lumps.NewLeaves(f, infos[lumps.LumpLeaves])             //
	entities, _ := lumps.NewEntities(f, infos[lumps.LumpEntities])       //

	// 3. Caricamento MarkSurfaces (Indirezione Leaf -> Face)
	if err := lumps.Seek(f, infos[lumps.LumpMarkSurfaces].Filepos); err != nil {
		return nil, err
	}
	markCount := int(infos[lumps.LumpMarkSurfaces].Size) / 2
	markSurfaces := make([]uint16, markCount)
	binary.Read(f, binary.LittleEndian, markSurfaces)

	player := &config.ConfigPlayer{}
	root := config.NewConfigRoot(nil, player, nil, 1.0, true, nil)

	// 4. Parsing Entità: Luci Globali e Posizione Player
	for _, ent := range entities {
		classname := ent.Properties["classname"]

		// Player Start
		if classname == "info_player_start" {
			if origin, ok := ent.Properties["origin"]; ok {
				var x, y, z float64
				fmt.Sscanf(origin, "%f %f %f", &x, &y, &z)
				// Swap assi: Quake (Z up) -> Engine (Y up)
				player.Position = geometry.XY{X: x, Y: z}
			}
		}

		// Luci Dinamiche/Puntiformi (Collegate alla Root)
		if classname == "light" || classname == "worldspawn" {
			intensity := 0.5
			if l, ok := ent.Properties["light"]; ok {
				if val, err := strconv.ParseFloat(l, 64); err == nil {
					intensity = val / 255.0
				}
			}

			kind := config.LightKindSpot
			pos := geometry.XYZ{}
			if classname == "worldspawn" {
				kind = config.LightKindAmbient
			} else if origin, ok := ent.Properties["origin"]; ok {
				var x, y, z float64
				fmt.Sscanf(origin, "%f %f %f", &x, &y, &z)
				pos = geometry.XYZ{X: x, Y: z, Z: -y}
			}

			root.Lights = append(root.Lights, config.NewConfigLight(pos, intensity, kind))
		}
	}

	// 5. Lowering: Conversione Leaves BSP -> ConfigVolume
	for leafIdx, leaf := range leaves {
		// Salta lo spazio solido (Foglia 0)
		if leafIdx == 0 {
			continue
		}

		volId := fmt.Sprintf("leaf_%d", leafIdx)
		volume := config.NewConfigVolume(volId, "quake_leaf")

		// Estrazione Facce della Foglia
		for i := uint16(0); i < leaf.NumFaces; i++ {
			faceIdx := markSurfaces[leaf.FirstFace+i]
			face := faces[faceIdx]
			texInfo := texInfos[face.TexInfo]

			// Risoluzione Texture/Tag
			tag := "default"
			if texInfo.MipTex < uint32(len(mipTextures)) && mipTextures[texInfo.MipTex] != nil {
				tag = mipTextures[texInfo.MipTex].Name
			}

			// Ricostruzione Geometria (Winding Order)
			var points []geometry.XYZ
			for j := uint16(0); j < face.NumEdges; j++ {
				se := surfEdges[face.FirstEdge+int32(j)]
				var v *lumps.Vertex

				if se >= 0 {
					v = vertexes[edges[se].Vertex0]
				} else {
					v = vertexes[edges[-se].Vertex1]
				}

				// Conversione coordinate spaziali (Z è l'altezza in Quake)
				points = append(points, geometry.XYZ{
					X: float64(v.X),
					Y: float64(v.Z),
					Z: float64(-v.Y),
				})
			}

			// Creazione Faccia
			material := config.NewConfigAnimation([]string{tag}, config.AnimationKindLoop, 1.0, 1.0)
			volume.Faces = append(volume.Faces, config.NewConfigFace(points, material, tag))
		}

		root.Volumes = append(root.Volumes, volume)
	}

	return root, nil
}
