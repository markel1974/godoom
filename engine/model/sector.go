package model

import "github.com/markel1974/godoom/engine/textures"

type Segment struct {
	Start  XY
	End    XY
	Ref    string
	Kind   int
	Sector *Sector
	Tag    string
}

func NewSegment(ref string, sector * Sector, kind int, start XY, end XY, tag string) *Segment{
	out := &Segment{
		Start:  start,
		End:    end,
		Ref:    ref,
		Kind:   kind,
		Sector: sector,
		Tag:    tag,
	}
	return out
}

func (k * Segment) Copy() * Segment {
	out := &Segment{
		Start:  k.Start,
		End:    k.End,
		Ref:    k.Ref,
		Kind:   k.Kind,
		Sector: k.Sector,
		Tag:    k.Tag,
	}
	return out
}

func (k * Segment) SetSector(ref string, sector * Sector) {
	k.Ref = ref
	k.Sector = sector
}




type Sector struct {
	Id           string
	Floor        float64
	Ceil         float64
	Segments     []*Segment
	Textures     bool
	Tag          string
	FloorTexture *textures.Texture
	CeilTexture  *textures.Texture
	UpperTexture *textures.Texture
	LowerTexture *textures.Texture
	WallTexture  *textures.Texture
	usage        int
	compileId    uint64
	references   map[string]bool
}

func NewSector(id string, segments []*Segment) *Sector {
	s := &Sector{
		Id:         id,
		Ceil:       0,
		Floor:      0,
		Segments:   segments,
		Textures:   false,
		usage:      0,
		compileId:  0,
		references: make(map[string]bool),
	}
	return s
}

func (s *Sector) Reference(compileId uint64) {
	if compileId != s.compileId {
		s.compileId = compileId
		s.usage = 0
		s.references = make(map[string]bool)
	} else {
		s.usage++
	}
}

func (s * Sector) GetCompileId() uint64{
	return s.compileId
}

func (s *Sector) GetUsage() int {
	return s.usage
}


func (s *Sector) Add(id string) {
	s.references[id] = true
}

func (s *Sector) Has(id string) bool {
	_, ok := s.references[id]
	return ok
}