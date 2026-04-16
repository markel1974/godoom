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

	// Inizializzazione Root con il riferimento al gestore texture popolato
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

	// 4. Conversione Geometria: BSP Leaves -> Volume
	for leafIdx, leaf := range leaves {
		// Saltiamo la leaf 0 (solitamente spazio solido esterno)
		if leafIdx == 0 {
			continue
		}

		volId := fmt.Sprintf("leaf_%d", leafIdx)
		volume := config.NewConfigVolume(volId, "quake_bsp")

		if leaf.NumFaces == 0 {
			// Un volume vuoto è normale per foglie che non contengono geometria visibile
			continue
		}

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

			// Ricostruzione Poligono 3D (Perimetro ordinato) dai SurfEdges
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

			// Creazione Animazione (Materiale) con l'ID texture registrato
			animKind := config.AnimationKindLoop
			if isSky {
				animKind = config.AnimationKindSky
			}
			material := config.NewConfigAnimation([]string{texName}, animKind, 1.0, 1.0)

			// TRIANGOLAZIONE CONVESSA: Bypassiamo la routine CDT complessa e preserviamo il Winding Order.
			// points contiene il poligono completo. triangles conterrà N-2 triangoli esatti.
			triangles := TriangulateConvex3d(points)

			// Inserimento rigoroso: 1 Face = 1 Triangolo
			for _, tri := range triangles {
				volume.Faces = append(volume.Faces, config.NewConfigFace(tri, material, texName))
			}
		}

		if len(volume.Faces) > 0 {
			root.Volumes = append(root.Volumes, volume)
		}
	}

	root.Player = config.NewConfigPlayer(playerPos, playerAngle, 8, 4, 20)
	root.Player.Speed = 400
	return root, nil
}

func CreateXYZ(x, y, z float64) geometry.XYZ {
	// Conversione coordinate: Quake Z-up -> Engine Z-up
	pos := geometry.XYZ{X: x, Y: z, Z: -y}
	//pos := geometry.XYZ{X: x, Y: y, Z: z}
	//fmt.Println("POS:", pos)
	return pos
}

/*
// TriangulateConvex3d esegue un rapido Triangle Fan invertito per OpenGL (CCW).
func TriangulateConvex3d(pts []XYZ) [][]XYZ {
	pLen := len(pts)
	if pLen < 3 {
		return nil
	}
	if pLen == 3 {
		// INVERTITO: da (0, 1, 2) a (0, 2, 1)
		return [][]XYZ{{pts[0], pts[2], pts[1]}}
	}

	output := make([][]XYZ, 0, pLen-2)
	for i := 1; i < pLen-1; i++ {
		// INVERTITO: pts[i+1] viene PRIMA di pts[i]
		output = append(output, []XYZ{pts[0], pts[i+1], pts[i]})
	}
	return output
}


*/

/*
// TriangulateConvex3d decompone un poligono convesso (es. Quake BSP) in triangoli.
// L'algoritmo a ventaglio preserva al 100% il Winding Order per il back-face culling.
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
*/

func TriangulateConvex3d(pts []geometry.XYZ) [][]geometry.XYZ {
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
