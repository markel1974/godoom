package script

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// scaleW defines the scaling factor for width calculations.
// scaleH defines the scaling factor for height calculations.
const (
	scaleW = 10.0
	scaleH = 50.0
)

// Builder is a type used to construct and configure complex objects or data structures.
type Builder struct {
}

// NewBuilder initializes and returns a new instance of Builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// Build processes the given data string to construct a Root configuration object, parsing vertices, sectors, and player info.
func (p *Builder) Build(id string) (*config.Root, error) {
	basePath := "resources" + string(os.PathSeparator) + "textures" + string(os.PathSeparator)
	t, tErr := NewTextures(basePath)
	if tErr != nil {
		return nil, tErr
	}

	player := config.NewConfigPlayer(geometry.XYZ{}, 0, 10, 90, 1.0, 20)
	player.Speed = 60

	cal := config.NewConfigCalibration(false, 0, 0, 0, 0, 0, 0, true)
	scaleFactor := geometry.XYZ{X: 1, Y: 1, Z: 1}
	cfg := config.NewConfigRoot(cal, nil, player, nil, scaleFactor, t)

	oldData := strings.Split(id, "\n")
	configSectorIdx := 0

	for _, data := range oldData {
		var verb string
		r := strings.NewReader(data)
		if _, err := fmt.Fscanf(r, "%s", &verb); err != nil {
			continue
		}
		switch verb {
		case "#":
			//comment
		case "vertex":
			vertex, err := p.parseVertex(r)
			if err != nil {
				return nil, err
			}
			cfg.Vertices = append(cfg.Vertices, vertex...)
		case "sector":
			cs, err := p.parseSector(r, cfg.Vertices, configSectorIdx)
			if err != nil {
				return nil, err
			}
			configSectorIdx++
			cfg.Sectors = append(cfg.Sectors, cs)
		case "player":
			if err := p.parsePlayer(r, cfg.Player); err != nil {
				return nil, err
			}
		default:
			continue
		}
	}

	if cfg.Vertices == nil {
		return nil, errors.New("nil vertices")
	}

	p.finalize(cfg)

	return cfg, nil
}

// finalize processes and updates sector segment relationships within the configuration by performing a topological rescan.
func (p *Builder) finalize(cfg *config.Root) {
	// 2. Fase di Sigillatura (Rescan Topologico)
	type edgeKey struct{ p1, p2 geometry.XY }
	lineDefsCache := make(map[edgeKey]*config.Sector)
	for _, sector := range cfg.Sectors {
		for _, s := range sector.Segments {
			lineDefsCache[edgeKey{s.Start, s.End}] = sector
		}
	}
	for _, sector := range cfg.Sectors {
		for _, s := range sector.Segments {
			if s.Kind != config.SegmentWall {
				// Cerchiamo il segmento invertito (il "lato B" della linea)
				revKey := edgeKey{p1: s.End, p2: s.Start}
				if neighborSector, ok := lineDefsCache[revKey]; ok {
					s.Tag = neighborSector.Id
					s.Kind = config.SegmentUnknown
				} else {
					// Nessun segmento corrispondente trovato: la linea deve essere un muro
					s.Kind = config.SegmentWall
					s.Tag = "unknown"
				}
			}
		}
	}
}

// parseVertex reads vertex data from an io.Reader, parsing it into a slice of geometry.XY points. Returns an error on failure.
func (p *Builder) parseVertex(r io.Reader) ([]geometry.XY, error) {
	var cfgVertices []geometry.XY
	for {
		var vertexY float64
		if _, err := fmt.Fscanf(r, "%f", &vertexY); err != nil {
			if err != io.EOF {
				return nil, err
			}
			break
		}
		for {
			xy := geometry.XY{Y: vertexY}
			if _, err := fmt.Fscanf(r, "%f", &xy.X); err != nil {
				if err != io.EOF {
					return nil, err
				}
				break
			}
			cfgVertices = append(cfgVertices, xy)
		}
	}
	return cfgVertices, nil
}

// parseSector parses sector data from the provided io.Reader and constructs a Sector object using given vertices and index.
// Returns the constructed Sector or an error if parsing fails.
func (p *Builder) parseSector(r io.Reader, cfgVertices []geometry.XY, configSectorIdx int) (*config.Sector, error) {
	if cfgVertices == nil {
		return nil, errors.New("nil vertices")
	}
	const falloff = 10.0
	const lightIntensity = 1.5
	cs := config.NewConfigSector(strconv.Itoa(configSectorIdx), lightIntensity, config.LightKindAmbient, falloff)
	if _, err := fmt.Fscanf(r, "%f%f", &cs.FloorY, &cs.CeilY); err != nil {
		return nil, err
	}
	type data struct {
		Val  int
		Kind int
	}
	var numbers []data
	var word string
	for {
		if _, err := fmt.Fscanf(r, "%32s", &word); err != nil {
			break
		}
		var d data
		d.Val = config.SegmentUnknown
		d.Kind = config.SegmentUnknown
		if word[0] != 'x' {
			if val, err := strconv.Atoi(word); err == nil {
				d.Val = val
			}
		}
		numbers = append(numbers, d)
	}

	if len(numbers) == 0 || len(numbers)%2 > 0 {
		return nil, errors.New("empty or invalid sector definition")
	}

	m := len(numbers) / 2
	for idx := 0; idx < m; idx++ {
		v1Idx := numbers[idx]
		v2Idx := numbers[(idx+1)%m] // Chiusura topologica
		neighborId := numbers[idx+m]

		if v1Idx.Val < 0 || v1Idx.Val >= len(cfgVertices) || v2Idx.Val < 0 || v2Idx.Val >= len(cfgVertices) {
			return nil, fmt.Errorf("invalid vertex index, max: %d", len(cfgVertices))
		}

		start := cfgVertices[v1Idx.Val]
		end := cfgVertices[v2Idx.Val]

		seg := config.NewConfigSegment("", neighborId.Kind, start, end)
		seg.Middle = config.NewConfigMaterial([]string{"wall2.ppm"}, config.MaterialKindLoop, scaleW, scaleH, 0, 0)
		seg.Lower = config.NewConfigMaterial([]string{"wall.ppm"}, config.MaterialKindLoop, scaleW, scaleH, 0, 0)
		seg.Upper = config.NewConfigMaterial([]string{"wall3.ppm"}, config.MaterialKindLoop, scaleW, scaleH, 0, 0)
		cs.Segments = append(cs.Segments, seg)
	}
	cs.Floor = config.NewConfigMaterial([]string{"floor.ppm"}, config.MaterialKindLoop, scaleW, scaleH, 0, 0)
	cs.Ceil = config.NewConfigMaterial([]string{"ceil.ppm"}, config.MaterialKindLoop, scaleW, scaleH, 0, 0)

	return cs, nil
}

// parsePlayer reads position and angle data from the given reader and populates the specified Player structure.
func (p *Builder) parsePlayer(r io.Reader, player *config.Player) error {
	if _, err := fmt.Fscanf(r, "%f %f %f", &player.Position.X, &player.Position.Y, &player.Angle); err != nil {
		return err
	}
	return nil
}
