package lumps

import (
	"archive/zip"
	"bytes"
	"fmt"
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
	PatchSize   [2]int32 // [0]=Width, [1]=Height (numero di control points)
}

type q3Vertex struct {
	Position  [3]float32
	TexCoord  [2]float32
	LMapCoord [2]float32
	Normal    [3]float32
	Color     [4]uint8
}

// evalBezier calcola la coordinata 1D lungo la curva di grado 2 per il fattore t [0.0 - 1.0]
func evalBezier(p0, p1, p2 float32, t float32) float32 {
	u := 1.0 - t
	return (u * u * p0) + (2.0 * u * t * p1) + (t * t * p2)
}

// tessellatePatch espande una griglia di punti di controllo 3x3 in una lista di triangoli planari.
// level specifica la risoluzione della tassellatura (es. 5 = 5x5 quad per patch).
func tessellatePatch(cp []q3Vertex, level int) []geometry.XYZ {
	var points []geometry.XYZ
	step := 1.0 / float32(level)

	// Pre-alloca la griglia di vertici interpolati
	L := level + 1
	grid := make([]geometry.XYZ, L*L)

	for i := 0; i <= level; i++ {
		tV := float32(i) * step
		for j := 0; j <= level; j++ {
			tU := float32(j) * step

			// Interpolazione asse U per le 3 righe
			var p [3]geometry.XYZ
			for row := 0; row < 3; row++ {
				idx := row * 3
				p[row] = CreateXYZ(
					float64(evalBezier(cp[idx].Position[0], cp[idx+1].Position[0], cp[idx+2].Position[0], tU)),
					float64(evalBezier(cp[idx].Position[1], cp[idx+1].Position[1], cp[idx+2].Position[1], tU)),
					float64(evalBezier(cp[idx].Position[2], cp[idx+1].Position[2], cp[idx+2].Position[2], tU)),
				)
			}

			// Interpolazione asse V sulla curva ricavata
			grid[i*L+j] = CreateXYZ(
				float64(evalBezier(float32(p[0].X), float32(p[1].X), float32(p[2].X), tV)),
				float64(evalBezier(float32(p[0].Y), float32(p[1].Y), float32(p[2].Y), tV)),
				float64(evalBezier(float32(p[0].Z), float32(p[1].Z), float32(p[2].Z), tV)),
			)
		}
	}

	// Estrazione dei triangoli (Triangle Strip to Triangle List)
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

// Pk3 implementa IReader per gli archivi idTech 3.
type Pk3 struct {
	archive *zip.ReadCloser
}

func NewPk3() *Pk3 {
	return &Pk3{}
}

func (p *Pk3) Setup(path string) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	p.archive = r
	return nil
}

func (p *Pk3) Open(path string) (io.ReadSeeker, error) {
	target := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	for _, f := range p.archive.File {
		if strings.ToLower(f.Name) == target {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			// Trasformazione stream in seeker
			data, err := io.ReadAll(rc)
			if err != nil {
				return nil, err
			}
			return bytes.NewReader(data), nil
		}
	}
	return nil, fmt.Errorf("file %s non trovato nel pk3", path)
}
