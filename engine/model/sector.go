package model

import (
	"encoding/json"

	"github.com/markel1974/godoom/engine/textures"
)

// Sector represents a 3D space in a model, defined by its boundaries, textures, and associated segments.
type Sector struct {
	ModelId      uint16
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
	references   map[uint64]bool

	VisibleSpans  [][2]float64
	LastCompileId uint64
}

// NewSector creates and initializes a new Sector instance with the given model ID, identifier, and segment list.
func NewSector(modelId uint16, id string, segments []*Segment) *Sector {
	s := &Sector{
		ModelId:    modelId,
		Id:         id,
		Ceil:       0,
		Floor:      0,
		Segments:   segments,
		Textures:   false,
		usage:      0,
		compileId:  0,
		references: make(map[uint64]bool),
	}
	return s
}

// Reference updates the Sector's compileId and resets usage and references if the given compileId is different.
func (s *Sector) Reference(compileId uint64) {
	if compileId != s.compileId {
		s.compileId = compileId
		s.usage = 0
		s.references = make(map[uint64]bool)
	} else {
		s.usage++
	}
}

// GetCompileId retrieves the current compileId associated with the Sector.
func (s *Sector) GetCompileId() uint64 {
	return s.compileId
}

// GetUsage returns the current usage count of the Sector.
func (s *Sector) GetUsage() int {
	return s.usage
}

// Add adds the given ID to the Sector's references map, marking it as referenced.
func (s *Sector) Add(id uint64) {
	s.references[id] = true
}

// Has checks if the given id exists in the Sector's references map and returns true if it does, false otherwise.
func (s *Sector) Has(id uint64) bool {
	_, ok := s.references[id]
	return ok
}

func (s *Sector) Print(indent bool) string {
	type printerSegment struct {
		Start XY
		End   XY
		Ref   string
		Kind  int
		Tag   string
	}
	type printerSector struct {
		ModelId  uint16
		Id       string
		Floor    float64
		Ceil     float64
		Segments []*printerSegment
		Textures bool
	}

	p := printerSector{ModelId: s.ModelId, Id: s.Id, Floor: s.Floor, Ceil: s.Ceil}
	for _, z := range s.Segments {
		ps := &printerSegment{Start: z.Start, End: z.End, Ref: z.Ref, Kind: z.Kind, Tag: z.Tag}
		p.Segments = append(p.Segments, ps)
	}
	p.Textures = s.Textures
	if indent {
		d, _ := json.MarshalIndent(p, "", "  ")
		return string(d)
	}
	d, _ := json.Marshal(p)
	return string(d)
}

func (s *Sector) IsVisible(x1 float64, x2 float64, id uint64) bool {
	if s.LastCompileId != id {
		s.VisibleSpans = s.VisibleSpans[:0]
		s.LastCompileId = id
	}
	for _, span := range s.VisibleSpans {
		if x1 >= span[0] && x2 <= span[1] {
			return false // Già coperto da un portale più grande
		}
	}
	return true
}
