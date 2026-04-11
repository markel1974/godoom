package quake

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strconv"

	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

type Parser struct {
	resourcePath string
	texManager   *Textures
	scale        float64
}

func NewParser(resourcePath string, scale float64) *Parser {
	if scale <= 0 {
		scale = 1.0
	}
	return &Parser{
		resourcePath: resourcePath,
		texManager:   NewTextures(),
		scale:        scale,
	}
}

func (p *Parser) Parse(filename string) (*config.ConfigRoot, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// 1. Caricamento Metadati Header
	infos, err := lumps.NewLumpInfos(f)
	if err != nil {
		return nil, err
	}

	// 2. Caricamento Lumps Geometrici
	vertexes, _ := lumps.NewVertexes(f, infos[lumps.LumpVertexes])
	edges, _ := lumps.NewEdges(f, infos[lumps.LumpEdges])
	surfEdges, _ := lumps.NewSurfEdges(f, infos[lumps.LumpSurfEdges])
	faces, _ := lumps.NewFace(f, infos[lumps.LumpFaces])
	texInfos, _ := lumps.NewTexInfos(f, infos[lumps.LumpTexInfo])
	mipTextures, _ := lumps.NewMipTextures(f, infos[lumps.LumpTextures])
	leaves, _ := lumps.NewLeaves(f, infos[lumps.LumpLeaves])
	entities, _ := lumps.NewEntities(f, infos[lumps.LumpEntities])

	// --- INIEZIONE TEXTURE ---
	// Registriamo tutte le texture trovate nel BSP nel gestore globale (p.texManager)
	// Questo permette al Compiler di trovarle pronte durante la fase di Setup GPU.
	for _, mt := range mipTextures {
		if mt != nil && mt.Name != "" {
			// Register cercherà il file (es. mt.Name + ".png") nel basePath delle risorse
			if err = p.texManager.Register(p.resourcePath, mt.Name); err != nil {
				fmt.Printf("Warning: texture %s non trovata nel path %s\n", mt.Name, p.resourcePath)
			}
		}
	}

	// Indirezione Leaf -> Face (MarkSurfaces)
	if err = lumps.Seek(f, infos[lumps.LumpMarkSurfaces].Filepos); err != nil {
		return nil, err
	}
	markCount := int(infos[lumps.LumpMarkSurfaces].Size) / 2
	markSurfaces := make([]uint16, markCount)
	_ = binary.Read(f, binary.LittleEndian, markSurfaces)

	// Inizializzazione ConfigRoot con il riferimento al gestore texture popolato
	player := &config.ConfigPlayer{}
	root := config.NewConfigRoot(nil, player, nil, p.scale, true, p.texManager)

	// 3. Parsing delle Entità (Luci, Player, Monsters)
	for _, ent := range entities {
		classname := ent.Properties["classname"]
		var pos geometry.XYZ
		if origin, ok := ent.Properties["origin"]; ok {
			var x, y, z float64
			_, _ = fmt.Sscanf(origin, "%f %f %f", &x, &y, &z)
			// Conversione coordinate: Quake Z-up -> Engine Y-up
			pos = geometry.XYZ{X: x * p.scale, Y: z * p.scale, Z: -y * p.scale}
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
				}
			}
			root.Lights = append(root.Lights, config.NewConfigLight(pos, intensity*p.scale, config.LightKindSpot, falloff))
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
			faceIdx := markSurfaces[leaf.FirstFace+i]
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
				points = append(points, geometry.XYZ{
					X: float64(v.X) * p.scale,
					Y: float64(v.Z) * p.scale,
					Z: float64(-v.Y) * p.scale,
				})
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

/*
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
				points = append(points, geometry.XYZ{X: float64(v.X), Y: float64(v.Z), Z: float64(-v.Y)})
			}

			// Creazione Faccia
			material := config.NewConfigAnimation([]string{tag}, config.AnimationKindLoop, 1.0, 1.0)
			volume.Faces = append(volume.Faces, config.NewConfigFace(points, material, tag))
		}

		root.Volumes = append(root.Volumes, volume)
	}

	return root, nil
}

*/
