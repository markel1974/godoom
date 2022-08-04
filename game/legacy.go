package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func ParseLegacyData(id string) (*Config, error) {
	var cfgVertices []XY
	cfg := &Config{
		Sectors: nil,
		Player:  &ConfigPlayer{},
	}

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
					xy := XY{Y: vertexY}
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
			cs := &ConfigSector{}
			cs.Id = strconv.Itoa(configSectorIdx)
			configSectorIdx++
			_, err := fmt.Fscanf(r, "%f%f", &cs.Floor, &cs.Ceil)
			if err != nil {
				return nil, err
			}
			var numbers []int
			for {
				if _, err := fmt.Fscanf(r, "%32s", &word); err != nil {
					break
				}
				var val int
				if word[0] != 'x' {
					val, _ = strconv.Atoi(word)
				} else {
					val = -1
				}
				numbers = append(numbers, val)
			}
			if len(numbers) == 0 || len(numbers)%2 > 0 {
				return nil, errors.New("empty sector number")
			}
			//numbers viene diviso a metà perche la prima parte contiene i riferimenti ai vertici e la seconda i Neighbors
			m := len(numbers) / 2
			for idx := 0; idx < m; idx++ {
				vertexId := numbers[idx]
				neighborId := numbers[idx+m]
				if vertexId < 0 || vertexId >= len(cfgVertices) {
					return nil, errors.New(fmt.Sprintf("invalid vertex number: %d max: %d", vertexId, len(cfgVertices)))
				}
				neighbor := &ConfigNeighbor{
					ConfigXY: ConfigXY{X: cfgVertices[vertexId].X, Y: cfgVertices[vertexId].Y},
					Id:       strconv.Itoa(neighborId),
				}
				cs.Neighbors = append(cs.Neighbors, neighbor)
				cs.WallTexture = "wall2.ppm"
				cs.LowerTexture = "wall.ppm"
				cs.UpperTexture = "wall3.ppm"
				cs.FloorTexture = "floor.ppm"
				cs.CeilTexture = "ceil.ppm"
				cs.Textures = true
			}
			cfg.Sectors = append(cfg.Sectors, cs)

		case "light":
			l := &ConfigLight{}
			_, _ = fmt.Fscanf(r, "%f %f %f %s %f %f %f", &l.Where.X, &l.Where.Z, &l.Where.Y, &l.Sector, &l.Light.X, &l.Light.Y, &l.Light.Z)
			cfg.Lights = append(cfg.Lights, l)

		case "player":
			_, _ = fmt.Fscanf(r, "%f %f %f %s", &cfg.Player.Position.X, &cfg.Player.Position.Y, &cfg.Player.Angle, &cfg.Player.Sector)

		default:
			continue
		}
	}

	if cfgVertices == nil {
		return nil, errors.New(fmt.Sprintf("nil vertices"))
	}

	out, _ := json.MarshalIndent(cfg, "", " ")
	fmt.Println(string(out))

	/*
		var ranges  = []int{ 3, 14, 27, 45 }
		for _, r := range ranges {
			out, _ := json.MarshalIndent(cfg.Vertices[r], "", " ")
			fmt.Println(string(out))
		}
	*/

	return cfg, nil
}

const stubOldEmpty = `
# Vertexes (Y coordinate, followed by list of X coordinates)
# -1 means wall
vertex	0	0 128
vertex  128  0 128

sector	0 40    0 1 3 2 -1 -1 -1 -1

player	5 5	0	0
`

const stubOld = `
vertex	0	0 6 28
vertex	2	1 17.5
vertex	5	4 6 18 21
vertex	6.5	9 11 13 13.5 17.5
vertex	7	5 7 8 9 11 13 13.5 15 17 19 21
vertex	7.5	4 6
vertex	10.5	4 6
vertex	11	5 7 8 9 11 13 13.5 15 17 19 21
vertex	11.5	9 11 13 13.5 17.5
vertex	13	4 6 18 21
vertex	16	1 17.5
vertex	18	0 6 28

# bottom floor
sector	0 20	 3 14 29 49             -1 1 11 22 
sector	0 20	 17 15 14 3 9           -1 12 11 0 21 
sector	0 20	 41 42 43 44 50 49 40   -1 20 -1 3 -1 -1 22

# tunnel under stairs
sector	0 14	 12 13 44 43 35 20      -1 21 -1 2 -1 4 

# hideout under stairs
sector	0 12	 16 20 35 31            -1 -1 3 -1 

# top floor
sector	16 28	 24 8 2 53 48 39        18 -1 7 -1 6 -1 
sector	16 28	 53 52 46 47 48         5 -1 8 10 -1 
sector	16 28	 1 2 8 7 6              23 -1 5 -1 10 

# top floor end, higher ceiling
sector	16 36	 46 52 51 45            -1 6 -1 24 
sector	16 36	 25 26 28 27            24 -1 10 -1

# top floor middle, lower ceiling
sector	16 26	 6 7 47 46 28 26        -1 7 -1 6 -1 9

# stairs
sector	2 20	 14 15 30 29            0 1 12 22 
sector	4 20	 15 17 32 30            11 1 13 22 
sector	6 20	 17 18 33 32            12 -1 14 -1 
sector	8 20	 18 19 34 33            13 19 15 20 
sector	10 24	 19 21 36 34            14 -1 16 -1 
sector	12 24	 21 22 37 36            15 -1 17 -1 
sector	14 28	 22 23 38 37            16 -1 18 -1 
sector	16 28	 23 24 39 38            17 -1 5 -1

# stairs windows
sector	8 14	 10 11 19 18            -1 21 -1 14 
sector	8 14	 33 34 42 41            -1 14 -1 2

sector	0 20	 4 13 12 11 10 9 3      -1 -1 3 -1 19 -1 1 
sector	0 20	 29 30 32 40 49         0 11 12 -1 2
sector	16 36	 1 6 5 0                -1 7 -1 24 
sector	16 36	 0 5 25 27 45 51        -1 23 -1 9 -1 8 

player	2 6	0	0
`

const stubOld2 = `
# Vertexes (Y coordinate, followed by list of X coordinates)
vertex	0	0 6 28
vertex	2	1 17.5
vertex	5	4 6 18 21
vertex	6.5	9 11 13 13.5 17.5
vertex	7	5 7 9 11 13 15 17 19 21
vertex	7.5	4 6
vertex	10.5	4 6
vertex	11	5 7 9 11 13 15 17 19 21
vertex	11.5	9 11 13 13.5 17.5
vertex	13	4 6 18 21
vertex	16	1 17.5
vertex	18	0 6 28
# 50
vertex 7  13.5 8
vertex 11 13.5 8

# Sectors (floor height, ceiling height, then vertex numbers in clockwise order)
# After the list of vertexes, comes the list of sector numbers on the "opposite" side of that wall; "x" = none.

# bottom floor
sector	 0 20	3 14 27 45	x x x x
sector	 0 20	3 4 13 12 11  10 9 16 15 14	x x x x x  x x x x x
sector	 0 20	27 28 29 36 37  38 39 40 46 45	x x x x x  x x x x x

# tunnel under stairs
sector	 0 14	12 13 40 39 52 50	x x x x x x

# hideout under stairs
sector	 0 12	51 50 52 53	x x x x

# top floor
sector	16 28	22 8 2 49 44 35		x x x x x x
sector	16 28	49 48 42 43 44		x x x x x
sector	16 28	1 2 8 7 6	x x x x x

# top floor end, higher ceiling
sector	16 36	0 1 6 5 23  25 41 42  48 47	x x x x x  x x x x x
sector	16 36	23 24 26 25	x x x x
# top floor middle, lower ceiling
sector	16 26	6 7 43  42 26 24	x x x x x x

# stairs
sector	 2 20	14 15 28 27	x x x x
sector	 4 20	15 16 29 28	x x x x
sector	 6 20	16 17 30 29	x x x x
sector	 8 20	17 18 31 30	x x x x
sector	10 24	18 19 32 31	x x x x
sector	12 24	19 20 33 32	x x x x
sector	14 28	20 21 34 33	x x x x
sector	16 28	21 22 35 34	x x x x

# stairs windows
sector	 8 14	10 11 18 17	x x x x
sector	 8 14	30 31 38 37	x x x x

# player: Location (x y), angle, and sector number
#player	2 9	0	0
player 2.3 6 0.4 0



# Light sources: Location (x z y; xz = 2d coordinates, y = height), sector number, color (r g b)
light	 2  4 18	0	247 228 170	# left corner of beginning
light	 2 14 18	0	247 228 170	# right corner of beginning
light	 9  9 11	4	247 170 112	# in hideout, redder
light	26  3 26	5	183 204 218	# cold in upstairs, back left corner
light	26 15 26	5	183 204 218	# cold in upstairs, back right corner
# A row of small lights upstairs in the back
light	1.2  3 30	24	247 228 170
light	1.2  7 30	24	247 228 170
light	1.2 11 30	24	247 228 170
light	1.2 15 30	24	247 228 170
`

const stubTest = `
# Vertexes (Y coordinate, followed by list of X coordinates)
vertex	0	0 6 28
vertex	2	1 17.5
vertex	5	4 6 18 21
vertex	6.5	9 11 13 13.5 17.5
vertex	7	5 7 9 11 13 15 17 19 21
vertex	7.5	4 6
vertex	10.5	4 6
vertex	11	5 7 9 11 13 15 17 19 21
vertex	11.5	9 11 13 13.5 17.5
vertex	13	4 6 18 21
vertex	16	1 17.5
vertex	18	0 6 28
# 50
vertex 7  13.5 8
vertex 11 13.5 8

# Sectors (floor height, ceiling height, then vertex numbers in clockwise order)
# After the list of vertexes, comes the list of sector numbers on the "opposite" side of that wall; "x" = none.

# bottom floor
sector	 0 20	3 14 27 45	x x x x
#sector	 0 20	3 4 13 12 11  10 9 16 15 14	x x x x x  x x x x x
#sector	 0 20	27 28 29 36 37  38 39 40 46 45	x x x x x  x x x x x

player 2.3 6 0.4 0
`
