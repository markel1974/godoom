package lumps

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	"io"
	"strings"

	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

const (
	LumpQ3Entities    = 0
	LumpQ3Textures    = 1
	LumpQ3Planes      = 2
	LumpQ3Nodes       = 3
	LumpQ3Leafs       = 4
	LumpQ3LeafFaces   = 5
	LumpQ3LeafBrushes = 6
	LumpQ3Models      = 7
	LumpQ3Brushes     = 8
	LumpQ3BrushSides  = 9
	LumpQ3Vertexes    = 10
	LumpQ3MeshVerts   = 11
	LumpQ3Effects     = 12
	LumpQ3Faces       = 13
	LumpQ3Lightmaps   = 14
	LumpQ3LightVols   = 15
	LumpQ3VisData     = 16
	NumQ3Lumps        = 17
)

type HeaderQ3 struct {
	Magic   [4]byte
	Version int32
	Lumps   [NumQ3Lumps]struct {
		Offset int32
		Length int32
	}
}

type q3Texture struct {
	Name     [64]byte
	Flags    uint32
	Contents uint32
}

type q3Model struct {
	Mins       [3]float32
	Maxs       [3]float32
	FirstFace  int32
	NumFaces   int32
	FirstBrush int32
	NumBrushes int32
}

type q3Face struct {
	TextureID   int32
	Effect      int32
	Type        int32 // 1=Polygon, 2=Patch, 3=Mesh, 4=Billboard
	VertexStart int32
	NumVertexes int32
	MeshStart   int32
	NumMesh     int32
	LightmapID  int32
	LMapCorner  [2]int32
	LMapSize    [2]int32
	LMapOrigin  [3]float32
	LMapVecs    [2][3]float32
	Normal      [3]float32
	PatchSize   [2]int32
}

type q3Vertex struct {
	Position  [3]float32
	TexCoord  [2]float32
	LMapCoord [2]float32
	Normal    [3]float32
	Color     [4]uint8
}

// Q3BSPReader analizza le mappe in formato idTech 3 (Quake 3 / Return to Castle Wolfenstein)
type Q3BSPReader struct {
	fs         IReader
	header     HeaderQ3
	rs         io.ReadSeeker
	texManager *Textures
}

func NewQ3BSPReader(rs io.ReadSeeker) *Q3BSPReader {
	return &Q3BSPReader{
		rs:         rs,
		texManager: NewTextures(),
	}
}

func (q3 *Q3BSPReader) Setup(r IReader) error {
	q3.fs = r
	if _, err := q3.rs.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err := binary.Read(q3.rs, binary.LittleEndian, &q3.header); err != nil {
		return err
	}
	if string(q3.header.Magic[:]) != "IBSP" || q3.header.Version != 46 {
		return fmt.Errorf("formato Quake 3 non valido (Magic: %s, Versione: %d)", string(q3.header.Magic[:]), q3.header.Version)
	}
	return nil
}

func (q3 *Q3BSPReader) GetEntities() ([]*Entity, error) {
	lump := q3.header.Lumps[LumpQ3Entities]
	if _, err := q3.rs.Seek(int64(lump.Offset), io.SeekStart); err != nil {
		return nil, err
	}
	data := make([]byte, lump.Length)
	if _, err := q3.rs.Read(data); err != nil {
		return nil, err
	}
	return parseEntityText(FromNullTerminatingString(data))
}

func (q3 *Q3BSPReader) GetModels() ([]*Model, error) {
	return nil, fmt.Errorf("implement me per i sub-models di Q3")
}

func (q3 *Q3BSPReader) RegisterPixels(name string, width, height int, indices []byte, isTransparent bool, transIndex byte, invertY bool) error {
	return nil // In Q3 le texture sono solitamente .tga o .jpg lette dal VFS nativamente come RGBA, il manager andrà adattato
}

func (q3 *Q3BSPReader) GetTextures() *Textures {
	return q3.texManager
}

// GetRawFaces risolve nativamente Index Buffer (MeshVerts) e Patch di Bezier
func (q3 *Q3BSPReader) GetRawFaces(modelIdx int) ([]*RawFace, error) {
	// 1. Lettura dei Modelli
	lModels := q3.header.Lumps[LumpQ3Models]
	q3.rs.Seek(int64(lModels.Offset), io.SeekStart)
	models := make([]q3Model, int(lModels.Length)/40)
	binary.Read(q3.rs, binary.LittleEndian, &models)

	if modelIdx < 0 || modelIdx >= len(models) {
		return nil, fmt.Errorf("modelIdx fuori range")
	}
	targetModel := models[modelIdx]

	// 2. Lettura massiva in RAM dei lump geometrici
	lFaces := q3.header.Lumps[LumpQ3Faces]
	q3.rs.Seek(int64(lFaces.Offset), io.SeekStart)
	faces := make([]q3Face, int(lFaces.Length)/104)
	binary.Read(q3.rs, binary.LittleEndian, &faces)

	lVerts := q3.header.Lumps[LumpQ3Vertexes]
	q3.rs.Seek(int64(lVerts.Offset), io.SeekStart)
	vertexes := make([]q3Vertex, int(lVerts.Length)/44)
	binary.Read(q3.rs, binary.LittleEndian, &vertexes)

	lMeshVerts := q3.header.Lumps[LumpQ3MeshVerts]
	q3.rs.Seek(int64(lMeshVerts.Offset), io.SeekStart)
	meshVerts := make([]int32, int(lMeshVerts.Length)/4)
	binary.Read(q3.rs, binary.LittleEndian, &meshVerts)

	lTextures := q3.header.Lumps[LumpQ3Textures]
	q3.rs.Seek(int64(lTextures.Offset), io.SeekStart)
	textures := make([]q3Texture, int(lTextures.Length)/72)
	binary.Read(q3.rs, binary.LittleEndian, &textures)

	var rawFaces []*RawFace

	// 3. Risoluzione Topologica
	for i := int32(0); i < targetModel.NumFaces; i++ {
		face := faces[targetModel.FirstFace+i]
		tex := textures[face.TextureID]

		texNameBytes := make([]byte, 0, 64)
		for _, b := range tex.Name {
			if b == 0 {
				break
			}
			texNameBytes = append(texNameBytes, b)
		}
		texName := strings.ToLower(string(texNameBytes))
		isSky := (tex.Flags & 0x4) != 0 // SURF_SKY

		switch face.Type {
		case 1, 3: // Poligono Convesso (1) o Mesh Complessa (3)
			// Q3 usa l'indicizzazione per formare direttamente triangoli
			for j := int32(0); j < face.NumMesh; j += 3 {
				var tri []geometry.XYZ
				for k := int32(0); k < 3; k++ {
					vIdx := face.VertexStart + meshVerts[face.MeshStart+j+k]
					v := vertexes[vIdx]
					tri = append(tri, CreateXYZ(float64(v.Position[0]), float64(v.Position[1]), float64(v.Position[2])))
				}
				rawFaces = append(rawFaces, &RawFace{
					Points:  tri, // Il Builder non dovrà fare il Fan se riceve già 3 punti
					TexName: texName,
					IsSky:   isSky,
				})
			}

		case 2: // PATCH DI BEZIER (Biquadratica)
			w := int(face.PatchSize[0])
			h := int(face.PatchSize[1])

			// Le patch in Q3 sono griglie 3x3 unite. Troviamo quante sub-patch ci sono.
			numPatchesX := (w - 1) / 2
			numPatchesY := (h - 1) / 2

			for y := 0; y < numPatchesY; y++ {
				for x := 0; x < numPatchesX; x++ {
					var cp [9]q3Vertex
					for row := 0; row < 3; row++ {
						for col := 0; col < 3; col++ {
							cpIdx := face.VertexStart + int32((y*2+row)*w+(x*2+col))
							cp[row*3+col] = vertexes[cpIdx]
						}
					}

					// Livello di Tassellatura (LOD). 5 = Risoluzione standard.
					triangles := tessellatePatch(cp, 5)

					// Raggruppiamo i punti a gruppi di 3 per formare i RawFace
					for t := 0; t < len(triangles); t += 3 {
						rawFaces = append(rawFaces, &RawFace{
							Points:  []geometry.XYZ{triangles[t], triangles[t+1], triangles[t+2]},
							TexName: texName,
							IsSky:   isSky,
						})
					}
				}
			}
		}
	}

	q3.compileTextures(rawFaces)

	return rawFaces, nil
}

// compileTextures cerca e decodifica i file JPEG associati alle facce estratte
func (q3 *Q3BSPReader) compileTextures(faces []*RawFace) {
	uniqueTextures := make(map[string]bool)
	for _, f := range faces {
		uniqueTextures[f.TexName] = true
	}

	for texName := range uniqueTextures {
		if texName == "noshader" || len(texName) == 0 {
			continue
		}

		// idTech 3 non specifica l'estensione nel BSP. Tentiamo il JPEG.
		jpgPath := texName + ".jpg"

		// fs è l'IReader (es. Pk3Reader) iniettato tramite Setup()
		file, err := q3.fs.Open(jpgPath)
		if err != nil {
			// Opzionale: implementare qui il fallback per la ricerca di ".tga"
			fmt.Printf("Warning: asset mancante %s\n", jpgPath)
			continue
		}

		// Decodifica lo stream JPEG
		img, _, err := image.Decode(file)
		if err != nil {
			fmt.Printf("Warning: decodifica fallita per %s: %v\n", jpgPath, err)
			continue
		}

		// Normalizzazione in memoria spaziale lineare a 32-bit (RGBA)
		bounds := img.Bounds()
		rgba := image.NewRGBA(bounds)
		draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

		// Invio del buffer [R,G,B,A, R,G,B,A...] al manager.
		// NOTA: Usa un metodo specifico per i 32-bit (es. RegisterPixelsRGBA)
		// bypassando la logica della palette a 8-bit usata in Q1/Q2.
		err = q3.texManager.RegisterPixelsRGBA(texName, bounds.Dx(), bounds.Dy(), rgba.Pix, false)

		if err != nil {
			fmt.Printf("Warning: registrazione texture fallita %s: %v\n", texName, err)
		}
	}
}

// evalBezier calcola la coordinata lungo la curva di grado 2 per il fattore t [0.0 - 1.0]
func evalBezier(p0, p1, p2 float32, t float32) float32 {
	u := 1.0 - t
	return (u * u * p0) + (2.0 * u * t * p1) + (t * t * p2)
}

// tessellatePatch espande i 9 punti di controllo in un array flat di triangoli
func tessellatePatch(cp [9]q3Vertex, level int) []geometry.XYZ {
	var points []geometry.XYZ
	step := 1.0 / float32(level)
	L := level + 1
	grid := make([]geometry.XYZ, L*L)

	// Calcolo interpolazione griglia
	for i := 0; i <= level; i++ {
		tV := float32(i) * step
		for j := 0; j <= level; j++ {
			tU := float32(j) * step
			var p [3]geometry.XYZ
			for row := 0; row < 3; row++ {
				idx := row * 3
				p[row] = CreateXYZ(
					float64(evalBezier(cp[idx].Position[0], cp[idx+1].Position[0], cp[idx+2].Position[0], tU)),
					float64(evalBezier(cp[idx].Position[1], cp[idx+1].Position[1], cp[idx+2].Position[1], tU)),
					float64(evalBezier(cp[idx].Position[2], cp[idx+1].Position[2], cp[idx+2].Position[2], tU)),
				)
			}
			grid[i*L+j] = CreateXYZ(
				float64(evalBezier(float32(p[0].X), float32(p[1].X), float32(p[2].X), tV)),
				float64(evalBezier(float32(p[0].Y), float32(p[1].Y), float32(p[2].Y), tV)),
				float64(evalBezier(float32(p[0].Z), float32(p[1].Z), float32(p[2].Z), tV)),
			)
		}
	}

	// Chiusura dei quadrati in triangoli (Winding Order CCW)
	for i := 0; i < level; i++ {
		for j := 0; j < level; j++ {
			v0 := grid[(i*L)+j]
			v1 := grid[(i*L)+j+1]
			v2 := grid[((i+1)*L)+j]
			v3 := grid[((i+1)*L)+j+1]

			points = append(points, v0, v2, v1)
			points = append(points, v1, v2, v3)
		}
	}
	return points
}
