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

// ParseScriptData parses the provided script data string and generates a ConfigRoot object or returns an error.
func ParseScriptData(id string) (*config.ConfigRoot, error) {
	var cfgVertices []geometry.XY
	basePath := "resources" + string(os.PathSeparator) + "textures" + string(os.PathSeparator)
	t, _ := NewTextures(basePath)
	cfg := config.NewConfigRoot(nil, &config.ConfigPlayer{}, nil, 1.0, false, t)
	//cfg := &model.ConfigRoot{
	//	Sectors: nil,
	//	ThingPlayer:  &model.ConfigPlayer{},
	//	Textures:
	//}

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
					//fmt.Printf("idx: %d, x: %f, y: %f\n", len(cfgVertices), xy.X, xy.Y)
					cfgVertices = append(cfgVertices, xy)
				}
			}

		case "sector":
			if cfgVertices == nil {
				return nil, errors.New(fmt.Sprintf("nil vertices"))
			}
			cs := config.NewConfigSector(strconv.Itoa(configSectorIdx), rnd.Float64(), config.LightKindAmbient)
			configSectorIdx++
			_, err := fmt.Fscanf(r, "%f%f", &cs.FloorY, &cs.CeilY)
			if err != nil {
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
				return nil, errors.New("empty sector number")
			}
			//numbers viene diviso a metà perche la prima parte contiene i riferimenti ai vertici e la seconda i Neighbors
			m := len(numbers) / 2
			for idx := 0; idx < m; idx++ {
				vertexId := numbers[idx]
				neighborId := numbers[idx+m]
				if vertexId.Val < 0 || vertexId.Val >= len(cfgVertices) {
					return nil, errors.New(fmt.Sprintf("invalid vertex number: %d max: %d", vertexId, len(cfgVertices)))
				}

				xy := geometry.XY{X: cfgVertices[vertexId.Val].X, Y: cfgVertices[vertexId.Val].Y}
				if idx == 0 {
					neighbor := config.NewConfigSegment("", neighborId.Kind, xy, xy, strconv.Itoa(neighborId.Val))
					neighbor.Middle = config.NewConfigAnimation([]string{"wall2.ppm"}, config.AnimationKindLoop, scaleW, scaleH)
					neighbor.Lower = config.NewConfigAnimation([]string{"wall.ppm"}, config.AnimationKindLoop, scaleW, scaleH)
					neighbor.Upper = config.NewConfigAnimation([]string{"wall3.ppm"}, config.AnimationKindLoop, scaleW, scaleH)
					cs.Segments = append(cs.Segments, neighbor)
				} else if idx == m-1 {
					prev := cs.Segments[idx-1]
					prev.End = xy
				} else {
					prev := cs.Segments[idx-1]
					prev.End = xy
					neighbor := config.NewConfigSegment("", config.DefinitionUnknown, xy, xy, "unknown")
					neighbor.Middle = config.NewConfigAnimation([]string{"wall2.ppm"}, config.AnimationKindLoop, scaleW, scaleH)
					neighbor.Lower = config.NewConfigAnimation([]string{"wall.ppm"}, config.AnimationKindLoop, scaleW, scaleH)
					neighbor.Upper = config.NewConfigAnimation([]string{"wall3.ppm"}, config.AnimationKindLoop, scaleW, scaleH)
					cs.Segments = append(cs.Segments, neighbor)
				}

				cs.Floor = config.NewConfigAnimation([]string{"floor.ppm"}, config.AnimationKindLoop, scaleW, scaleH)
				cs.Ceil = config.NewConfigAnimation([]string{"ceil.ppm"}, config.AnimationKindLoop, scaleW, scaleH)
				//cs.Textures = true
			}
			cfg.Sectors = append(cfg.Sectors, cs)

		case "light":
			//l := &model.ConfigLight{}
			//_, _ = fmt.Fscanf(r, "%f %f %f %s %f %f %f", &l.Where.X, &l.Where.Z, &l.Where.Y, &l.Sector, &l.Light.X, &l.Light.Y, &l.Light.Z)
			//cfg.Lights = append(cfg.Lights, l)

		case "player":
			_, _ = fmt.Fscanf(r, "%f %f %f %s", &cfg.Player.Position.X, &cfg.Player.Position.Y, &cfg.Player.Angle)

		default:
			continue
		}
	}

	if cfgVertices == nil {
		return nil, errors.New(fmt.Sprintf("nil vertices"))
	}

	//out, _ := json.MarshalIndent(cfg, "", " ")
	//fmt.Println(string(out))

	/*
		var ranges  = []int{ 3, 14, 27, 45 }
		for _, r := range ranges {
			out, _ := json.MarshalIndent(cfg.Vertices[r], "", " ")
			fmt.Println(string(out))
		}
	*/

	return cfg, nil
}
