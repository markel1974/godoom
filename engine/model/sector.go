package model

import "github.com/markel1974/godoom/engine/textures"

type Sector struct {
	Id           string
	Floor        float64
	Ceil         float64
	Vertices     []XY
	NPoints      uint64
	NeighborsIds []string
	//TODO REMOVE
	NeighborsRefs []int
	Neighbors     []*Sector
	Textures      bool
	FloorTexture  *textures.Texture
	CeilTexture   *textures.Texture
	UpperTexture  *textures.Texture
	LowerTexture  *textures.Texture
	WallTexture   *textures.Texture
	usage         int
	compileId     int
}

func NewSector(id string, nPoints uint64, vertices []XY, neighborsIds []string) *Sector {
	s := &Sector{
		Id:            id,
		Ceil:          0,
		Floor:         0,
		Vertices:      vertices,
		NeighborsRefs: nil,
		Neighbors:     nil,
		NeighborsIds:  neighborsIds,
		NPoints:       nPoints,
		Textures:      false,
		usage:         0,
		compileId:     0,
	}
	return s
}

func (s *Sector) Reference(compileId int) {
	if compileId != s.compileId {
		s.compileId = compileId
		s.usage = 0
	}
}

func (s *Sector) AddUsage() {
	s.usage++
}

func (s *Sector) GetUsage() int {
	return s.usage
}
