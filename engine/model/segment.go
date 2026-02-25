package model

// Segment represents a line segment in 2D space, defined by its start and end coordinates, and associated metadata.
type Segment struct {
	Start  XY
	End    XY
	Ref    string
	Kind   int
	Sector *Sector
	Tag    string
}

// NewSegment creates and returns a new Segment instance with specified start, end Points, reference, Kind, Sector, and tag.
func NewSegment(ref string, sector *Sector, kind int, start XY, end XY, tag string) *Segment {
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

// Copy creates and returns a deep copy of the Segment instance.
func (k *Segment) Copy() *Segment {
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

// SetSector assigns a reference string and associates the segment with a specified Sector.
func (k *Segment) SetSector(ref string, sector *Sector) {
	k.Ref = ref
	k.Sector = sector
}
