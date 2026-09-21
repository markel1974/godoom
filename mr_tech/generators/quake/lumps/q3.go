package lumps

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	"io"
	"strings"

	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/geometry"
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
	arc        IArchive
	header     HeaderQ3
	rs         io.ReadSeeker
	texManager *Textures
}

func NewQ3BSPReader(arc IArchive, rs io.ReadSeeker) *Q3BSPReader {
	return &Q3BSPReader{
		arc:        arc,
		rs:         rs,
		texManager: NewTextures(),
	}
}

func (q3 *Q3BSPReader) Setup() error {
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

// GetArchive returns the IArchive instance associated with the Q1BSPReader, used for file access and data retrieval.
func (q3 *Q3BSPReader) GetArchive() IArchive {
	return q3.arc
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
	return NewEntitiesFromText(FromNullTerminatingString(data))
}

func (q3 *Q3BSPReader) GetModels() ([]*Model, error) {
	lModels := q3.header.Lumps[LumpQ3Models]
	if _, err := q3.rs.Seek(int64(lModels.Offset), io.SeekStart); err != nil {
		return nil, err
	}

	numModels := int(lModels.Length) / 40
	models := make([]q3Model, numModels)
	if err := binary.Read(q3.rs, binary.LittleEndian, &models); err != nil {
		return nil, err
	}

	out := make([]*Model, numModels)
	for i, m := range models {
		out[i] = &Model{
			Mins:      m.Mins,
			Maxs:      m.Maxs,
			FirstFace: m.FirstFace,
			NumFaces:  m.NumFaces,
			// Q3 non usa Origin/HeadNode/VisLeafs nel lump Models, le collisioni
			// sono basate sui Brush associati (FirstBrush, NumBrushes).
		}
	}
	return out, nil
}

func (q3 *Q3BSPReader) RegisterPixels(name string, width, height int, indices []byte, isTransparent bool, transIndex byte, invertY bool) error {
	return nil // In Q3 le texture sono solitamente .tga o .jpg lette dal VFS nativamente come RGBA, il manager andrà adattato
}

func (q3 *Q3BSPReader) RegisterPixelsRGBA(name string, width, height int, pixels []byte, invertY bool) error {
	return q3.texManager.RegisterPixelsRGBA(name, width, height, pixels, invertY)
}

func (q3 *Q3BSPReader) GetTextures() *Textures {
	return q3.texManager
}

// GetRawFaces risolve nativamente Index Buffer (MeshVerts) e Patch di Bezier
func (q3 *Q3BSPReader) GetRawFaces(modelIdx int) ([]*RawFace, error) {
	lModels := q3.header.Lumps[LumpQ3Models]
	if _, err := q3.rs.Seek(int64(lModels.Offset), io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to models lump: %w", err)
	}
	models := make([]q3Model, int(lModels.Length)/40)
	if err := binary.Read(q3.rs, binary.LittleEndian, &models); err != nil {
		return nil, fmt.Errorf("failed to read models lump: %w", err)
	}

	if modelIdx < 0 || modelIdx >= len(models) {
		return nil, fmt.Errorf("modelIdx out of range")
	}
	targetModel := models[modelIdx]

	//Geometrical lumps
	lFaces := q3.header.Lumps[LumpQ3Faces]
	if _, err := q3.rs.Seek(int64(lFaces.Offset), io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to faces lump: %w", err)
	}
	faces := make([]q3Face, int(lFaces.Length)/104)
	if err := binary.Read(q3.rs, binary.LittleEndian, &faces); err != nil {
		return nil, fmt.Errorf("failed to read faces lump: %w", err)
	}

	lVerts := q3.header.Lumps[LumpQ3Vertexes]
	if _, err := q3.rs.Seek(int64(lVerts.Offset), io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to vertexes lump: %w", err)
	}
	vertexes := make([]q3Vertex, int(lVerts.Length)/44)
	if err := binary.Read(q3.rs, binary.LittleEndian, &vertexes); err != nil {
		return nil, fmt.Errorf("failed to read vertexes lump: %w", err)
	}

	lMeshVerts := q3.header.Lumps[LumpQ3MeshVerts]
	if _, err := q3.rs.Seek(int64(lMeshVerts.Offset), io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to mesh verts lump: %w", err)
	}
	meshVerts := make([]int32, int(lMeshVerts.Length)/4)
	if err := binary.Read(q3.rs, binary.LittleEndian, &meshVerts); err != nil {
		return nil, fmt.Errorf("failed to read mesh verts lump: %w", err)
	}

	lTextures := q3.header.Lumps[LumpQ3Textures]
	if _, err := q3.rs.Seek(int64(lTextures.Offset), io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to textures lump: %w", err)
	}
	textures := make([]q3Texture, int(lTextures.Length)/72)
	if err := binary.Read(q3.rs, binary.LittleEndian, &textures); err != nil {
		return nil, fmt.Errorf("failed to read textures lump: %w", err)
	}

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

		if (tex.Flags & 0x80) != 0 {
			continue // SURF_NODRAW
		}

		switch face.Type {
		case 1, 3: // Poligono Convesso (1) o Mesh Complessa (3)
			// Q3 usa l'indicizzazione per formare direttamente triangoli
			for j := int32(0); j < face.NumMesh; j += 3 {
				var tri []geometry.XYZ
				var uvs [][2]float64
				for k := int32(0); k < 3; k++ {
					vIdx := face.VertexStart + meshVerts[face.MeshStart+j+k]
					v := vertexes[vIdx]
					tri = append(tri, CreateXYZ(float64(v.Position[0]), float64(v.Position[1]), float64(v.Position[2])))
					uvs = append(uvs, [2]float64{float64(v.TexCoord[0]), float64(v.TexCoord[1])})
				}
				rawFaces = append(rawFaces, &RawFace{
					Points:  tri, // Il Builder non dovrà fare il Fan se riceve già 3 punti
					UVs:     uvs,
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
					triangles, uvs := q3.tessellatePatch(cp, 5)

					for t := 0; t < len(triangles); t += 3 {
						rawFaces = append(rawFaces, &RawFace{
							Points:  []geometry.XYZ{triangles[t], triangles[t+1], triangles[t+2]},
							UVs:     [][2]float64{uvs[t], uvs[t+1], uvs[t+2]},
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

		var img image.Image
		var err error

		// Prova prima con JPEG
		jpgPath := texName + ".jpg"
		file, errJpg := q3.arc.Open(jpgPath)
		if errJpg == nil {
			img, _, err = image.Decode(file)
		} else {
			// Fallback su TGA
			tgaPath := texName + ".tga"
			fileTga, errTga := q3.arc.Open(tgaPath)
			if errTga == nil {
				img, err = common.DecodeTGA(fileTga)
			} else {
				fmt.Printf("Warning: asset mancante %s (.jpg/.tga)\n", texName)
				continue
			}
		}

		if err != nil {
			fmt.Printf("Warning: decodifica fallita per %s: %v\n", texName, err)
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
func (q3 *Q3BSPReader) evalBezier(p0, p1, p2 float32, t float32) float32 {
	u := 1.0 - t
	return (u * u * p0) + (2.0 * u * t * p1) + (t * t * p2)
}

// tessellatePatch espande i 9 punti di controllo in un array flat di triangoli
func (q3 *Q3BSPReader) tessellatePatch(cp [9]q3Vertex, level int) ([]geometry.XYZ, [][2]float64) {
	var points []geometry.XYZ
	var uvs [][2]float64
	step := 1.0 / float32(level)
	L := level + 1
	grid := make([]geometry.XYZ, L*L)
	gridUV := make([][2]float64, L*L)

	// Calcolo interpolazione griglia
	for i := 0; i <= level; i++ {
		tV := float32(i) * step
		for j := 0; j <= level; j++ {
			tU := float32(j) * step
			var p [3]geometry.XYZ
			var puv [3][2]float32
			for row := 0; row < 3; row++ {
				idx := row * 3
				p[row] = CreateXYZ(
					float64(q3.evalBezier(cp[idx].Position[0], cp[idx+1].Position[0], cp[idx+2].Position[0], tU)),
					float64(q3.evalBezier(cp[idx].Position[1], cp[idx+1].Position[1], cp[idx+2].Position[1], tU)),
					float64(q3.evalBezier(cp[idx].Position[2], cp[idx+1].Position[2], cp[idx+2].Position[2], tU)),
				)
				puv[row] = [2]float32{
					q3.evalBezier(cp[idx].TexCoord[0], cp[idx+1].TexCoord[0], cp[idx+2].TexCoord[0], tU),
					q3.evalBezier(cp[idx].TexCoord[1], cp[idx+1].TexCoord[1], cp[idx+2].TexCoord[1], tU),
				}
			}
			grid[i*L+j] = CreateXYZ(
				float64(q3.evalBezier(float32(p[0].X), float32(p[1].X), float32(p[2].X), tV)),
				float64(q3.evalBezier(float32(p[0].Y), float32(p[1].Y), float32(p[2].Y), tV)),
				float64(q3.evalBezier(float32(p[0].Z), float32(p[1].Z), float32(p[2].Z), tV)),
			)
			gridUV[i*L+j] = [2]float64{
				float64(q3.evalBezier(puv[0][0], puv[1][0], puv[2][0], tV)),
				float64(q3.evalBezier(puv[0][1], puv[1][1], puv[2][1], tV)),
			}
		}
	}

	// Chiusura dei quadrati in triangoli (Winding Order CCW)
	for i := 0; i < level; i++ {
		for j := 0; j < level; j++ {
			idx0 := (i * L) + j
			idx1 := (i * L) + j + 1
			idx2 := ((i + 1) * L) + j
			idx3 := ((i + 1) * L) + j + 1

			points = append(points, grid[idx0], grid[idx2], grid[idx1])
			uvs = append(uvs, gridUV[idx0], gridUV[idx2], gridUV[idx1])

			points = append(points, grid[idx1], grid[idx2], grid[idx3])
			uvs = append(uvs, gridUV[idx1], gridUV[idx2], gridUV[idx3])
		}
	}
	return points, uvs
}

// GetExternalBModelFileName retrieves the file name of an external BModel based on the provided classname.
func (q3 *Q3BSPReader) GetExternalBModelFileName(classname string) string {
	return _q3DictBModel[classname]
}

// GetModelFileName returns the file name of the model associated with the specified classname.
func (q3 *Q3BSPReader) GetModelFileName(classname string) string {
	return _q3DictModelFilename[classname]
}
