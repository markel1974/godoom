package model

import "github.com/markel1974/godoom/engine/textures"

// PointInSegments Algoritmo Ray-Casting (Odd-Even) per i settori del Portal Engine
func PointInSegments(px float64, py float64, segments []*Segment) bool {
	nVert := len(segments)
	if nVert == 0 {
		return false
	}
	c := false
	j := nVert - 1
	for i := 0; i < nVert; i++ {
		pi := segments[i].Start
		pj := segments[j].Start
		// Inverte lo stato logico se il raggio proiettato lungo X attraversa l'edge
		if ((pi.Y > py) != (pj.Y > py)) && (px < (pj.X-pi.X)*(py-pi.Y)/(pj.Y-pi.Y)+pi.X) {
			c = !c
		}
		j = i
	}
	return c
}

// Segment represents a line segment in 2D space, defined by its start and end coordinates, and associated metadata.
type Segment struct {
	Start      XY
	End        XY
	Ref        string
	Kind       int
	Sector     *Sector
	Tag        string
	Animations *SegmentAnimations
}

// NewSegment creates and returns a new Segment instance with specified start, end Points, reference, Kind, Sector, and tag.
func NewSegment(ref string, sector *Sector, kind int, start XY, end XY, tag string, tUpper, tMiddle, tLower *textures.Animation) *Segment {
	out := &Segment{
		Start:      start,
		End:        end,
		Ref:        ref,
		Kind:       kind,
		Sector:     sector,
		Tag:        tag,
		Animations: NewSegmentAnimation(tUpper, tMiddle, tLower),
	}
	return out
}

// Copy creates and returns a deep copy of the Segment instance.
func (k *Segment) Copy() *Segment {
	out := &Segment{
		Start:      k.Start,
		End:        k.End,
		Ref:        k.Ref,
		Kind:       k.Kind,
		Sector:     k.Sector,
		Tag:        k.Tag,
		Animations: k.Animations.Clone(),
	}
	return out
}

// SetSector assigns a reference string and associates the segment with a specified Sector.
func (k *Segment) SetSector(ref string, sector *Sector) {
	k.Ref = ref
	k.Sector = sector
}
