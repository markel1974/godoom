package script

import (
	"errors"
	"fmt"
	"io"
	rnd "math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/engine/model"
)

// ParseScriptData parses the provided script data string and generates a ConfigRoot object or returns an error.
func ParseScriptData(id string) (*model.ConfigRoot, error) {
	var cfgVertices []model.XY
	basePath := "resources" + string(os.PathSeparator) + "textures" + string(os.PathSeparator)
	t, _ := NewTextures(basePath)
	cfg := model.NewConfigRoot(nil, &model.ConfigPlayer{}, nil, 1.0, false, t)
	//cfg := &model.ConfigRoot{
	//	Sectors: nil,
	//	Player:  &model.ConfigPlayer{},
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
					xy := model.XY{Y: vertexY}
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
			cs := model.NewConfigSector(strconv.Itoa(configSectorIdx))

			//todo light entry
			cs.Light.Intensity = rnd.Float64()
			cs.Light.Kind = "spot"
			configSectorIdx++
			_, err := fmt.Fscanf(r, "%f%f", &cs.Floor, &cs.Ceil)
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
					d.Val = model.DefinitionUnknown
					d.Kind = model.DefinitionUnknown
				} else {
					if val, err := strconv.Atoi(word); err != nil {
						d.Val = model.DefinitionUnknown
						d.Kind = model.DefinitionUnknown
					} else {
						d.Val = val
						d.Kind = model.DefinitionJoin
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

				xy := model.XY{X: cfgVertices[vertexId.Val].X, Y: cfgVertices[vertexId.Val].Y}
				if idx == 0 {
					neighbor := &model.ConfigSegment{Start: xy, End: xy, Neighbor: strconv.Itoa(neighborId.Val), Kind: neighborId.Kind}
					neighbor.TextureMiddle = append(neighbor.TextureMiddle, "wall2.ppm")
					neighbor.TextureLower = append(neighbor.TextureLower, "wall.ppm")
					neighbor.TextureUpper = append(neighbor.TextureUpper, "wall3.ppm")

					cs.Segments = append(cs.Segments, neighbor)
				} else if idx == m-1 {
					prev := cs.Segments[idx-1]
					prev.End = xy
				} else {
					prev := cs.Segments[idx-1]
					prev.End = xy
					neighbor := &model.ConfigSegment{Start: xy, End: xy, Neighbor: "unknown", Kind: model.DefinitionUnknown}
					neighbor.TextureMiddle = append(neighbor.TextureMiddle, "wall2.ppm")
					neighbor.TextureLower = append(neighbor.TextureLower, "wall.ppm")
					neighbor.TextureUpper = append(neighbor.TextureUpper, "wall3.ppm")
					cs.Segments = append(cs.Segments, neighbor)
				}

				cs.TextureFloor = append(cs.TextureFloor, "floor.ppm")
				cs.TextureCeil = append(cs.TextureCeil, "ceil.ppm")
				cs.TextureScaleFactor = 50.0
				//cs.Textures = true
			}
			cfg.Sectors = append(cfg.Sectors, cs)

		case "light":
			//l := &model.ConfigLight{}
			//_, _ = fmt.Fscanf(r, "%f %f %f %s %f %f %f", &l.Where.X, &l.Where.Z, &l.Where.Y, &l.Sector, &l.Light.X, &l.Light.Y, &l.Light.Z)
			//cfg.Lights = append(cfg.Lights, l)

		case "player":
			_, _ = fmt.Fscanf(r, "%f %f %f %s", &cfg.Player.Position.X, &cfg.Player.Position.Y, &cfg.Player.Angle, &cfg.Player.Sector)

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
