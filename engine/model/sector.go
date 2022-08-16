package model

import "github.com/markel1974/godoom/engine/textures"

type XYKind2 struct {
	XY
	Ref    string
	Kind   int
	Sector *Sector
}

func NewXYKind(ref string, sector * Sector, kind int, xy XY) *XYKind2{
	k := &XYKind2{}
	k.Update(ref, sector, kind, xy)
	return k
}

func (k * XYKind2) Update(ref string, sector * Sector, kind int, xy XY) {
	k.Ref = ref
	k.Sector = sector
	k.Kind = kind
	k.XY = xy
}

func (k * XYKind2) Clone() *XYKind2 {
	return NewXYKind(k.Ref, k.Sector, k.Kind, k.XY)
}



type Sector struct {
	Id           string
	Floor        float64
	Ceil         float64
	Vertices     []*XYKind2
	NPoints      uint64
	//Neighbors    []*Sector
	Textures     bool
	Tag          string
	FloorTexture *textures.Texture
	CeilTexture  *textures.Texture
	UpperTexture *textures.Texture
	LowerTexture *textures.Texture
	WallTexture  *textures.Texture
	usage        int
	compileId    int
}

func NewSector(id string, nPoints uint64, vertices []*XYKind2) *Sector {
	s := &Sector{
		Id:            id,
		Ceil:          0,
		Floor:         0,
		Vertices:      vertices,
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
