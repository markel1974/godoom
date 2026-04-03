package script

import (
	"errors"
	"fmt"
	"io"
	rnd "math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

const (
	scaleW = 10.0
	scaleH = 50.0
)

func ParseScriptData(id string) (*config.ConfigRoot, error) {
	var cfgVertices []geometry.XY
	basePath := "resources" + string(os.PathSeparator) + "textures" + string(os.PathSeparator)
	t, _ := NewTextures(basePath)
	cfg := config.NewConfigRoot(nil, &config.ConfigPlayer{}, nil, 1.0, false, t)

	oldData := strings.Split(id, "\n")
	configSectorIdx := 0

	for _, data := range oldData {
		var verb string
		var word string
		r := strings.NewReader(data)
		if _, err := fmt.Fscanf(r, "%s", &verb); err != nil {
			continue
		}
		switch verb {
		case "#":
			continue

		case "vertex":
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

		case "sector":
			if cfgVertices == nil {
				return nil, errors.New("nil vertices")
			}
			cs := config.NewConfigSector(strconv.Itoa(configSectorIdx), rnd.Float64(), config.LightKindAmbient)
			configSectorIdx++

			if _, err := fmt.Fscanf(r, "%f%f", &cs.FloorY, &cs.CeilY); err != nil {
				return nil, err
			}

			type data struct {
				Val  int
				Kind int
			}
			var numbers []data
			for {
				if _, err := fmt.Fscanf(r, "%32s", &word); err != nil {
					break
				}
				var d data
				if word[0] == 'x' {
					d.Val = config.DefinitionUnknown
					d.Kind = config.DefinitionUnknown
				} else {
					if val, err := strconv.Atoi(word); err != nil {
						d.Val = config.DefinitionUnknown
						d.Kind = config.DefinitionUnknown
					} else {
						d.Val = val
						d.Kind = config.DefinitionJoin
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

				seg := config.NewConfigSegment("", neighborId.Kind, start, end, strconv.Itoa(neighborId.Val))
				seg.Middle = config.NewConfigAnimation([]string{"wall2.ppm"}, config.AnimationKindLoop, scaleW, scaleH)
				seg.Lower = config.NewConfigAnimation([]string{"wall.ppm"}, config.AnimationKindLoop, scaleW, scaleH)
				seg.Upper = config.NewConfigAnimation([]string{"wall3.ppm"}, config.AnimationKindLoop, scaleW, scaleH)
				cs.Segments = append(cs.Segments, seg)
			}

			cs.Floor = config.NewConfigAnimation([]string{"floor.ppm"}, config.AnimationKindLoop, scaleW, scaleH)
			cs.Ceil = config.NewConfigAnimation([]string{"ceil.ppm"}, config.AnimationKindLoop, scaleW, scaleH)

			cfg.Sectors = append(cfg.Sectors, cs)

		case "player":
			_, _ = fmt.Fscanf(r, "%f %f %f %s", &cfg.Player.Position.X, &cfg.Player.Position.Y, &cfg.Player.Angle)

		default:
			continue
		}
	}

	// 2. Fase di Sigillatura (Rescan Topologico)
	type edgeKey struct{ p1, p2 geometry.XY }
	lineDefsCache := make(map[edgeKey]*config.ConfigSector)
	for _, sector := range cfg.Sectors {
		for _, s := range sector.Segments {
			lineDefsCache[edgeKey{s.Start, s.End}] = sector
		}
	}
	for _, sector := range cfg.Sectors {
		for _, s := range sector.Segments {
			if s.Kind != config.DefinitionWall {
				// Cerchiamo il segmento invertito (il "lato B" della linea)
				revKey := edgeKey{p1: s.End, p2: s.Start}
				if neighborSector, ok := lineDefsCache[revKey]; ok {
					s.Tag = neighborSector.Id
					s.Kind = config.DefinitionUnknown
				} else {
					// Nessun segmento corrispondente trovato: la linea deve essere un muro
					s.Kind = config.DefinitionWall
					s.Tag = "unknown"
				}
			}
		}
	}

	if cfgVertices == nil {
		return nil, errors.New("nil vertices")
	}

	cfg.Vertices = cfgVertices

	return cfg, nil
}
