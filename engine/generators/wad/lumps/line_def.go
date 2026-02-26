package lumps

import (
	"encoding/binary"
	"os"
	"strings"
	"unsafe"
)

// Impassible indicates that the area cannot be traversed.
// BlockMonsters blocks monsters from passing through.
// TwoSided denotes a two-sided line definition.
// UpperUnpegged signifies that the upper texture is not pegged.
// LowerUnpegged signifies that the lower texture is not pegged.
// Secret represents a secret area or line.
// BlockSound prevents sound from traveling through the line.
// NotOnMap marks the line as not visible on the automap.
// AlreadyOnMap indicates that the line is already displayed on the automap.
const (
	Impassible    = 0
	BlockMonsters = 1
	TwoSided      = 2
	UpperUnpegged = 3
	LowerUnpegged = 4
	Secret        = 5
	BlockSound    = 6
	NotOnMap      = 7
	AlreadyOnMap
)

// LineDef represents a line definition in a map, typically used to define walls, triggers, and sector boundaries.
// VertexStart specifies the starting vertex index of the line.
// VertexEnd specifies the ending vertex index of the line.
// Flags contains bitwise flags defining properties like impassibility and visibility.
// Function specifies the special function associated with the line for triggering events.
// Tag is a reference number used to associate the line with other map elements.
// SideDefRight specifies the index of the right-sided SideDef of the line or -1 if not present.
// SideDefLeft specifies the index of the left-sided SideDef of the line or -1 if not present.
type LineDef struct {
	VertexStart  int16
	VertexEnd    int16
	Flags        int16
	Function     int16
	Tag          int16
	SideDefRight int16
	SideDefLeft  int16
}

// HasFlag checks if a specific flag is set in the Flags field of the LineDef by performing a bitwise operation.
func (l *LineDef) HasFlag(flag int) bool {
	return l.Flags>>flag&1 == 1
}

// PrintBits returns a comma-separated string of flag names set in the LineDef.Flags field.
func (l *LineDef) PrintBits() string {
	var data []string
	if (l.Flags>>TwoSided)&1 == 1 {
		data = append(data, "twoSided")
	}
	if (l.Flags>>Impassible)&1 == 1 {
		data = append(data, "impassible")
	}
	if (l.Flags>>BlockMonsters)&1 == 1 {
		data = append(data, "blockMonsters")
	}
	if (l.Flags>>UpperUnpegged)&1 == 1 {
		data = append(data, "upperUnpegged")
	}
	if (l.Flags>>LowerUnpegged)&1 == 1 {
		data = append(data, "lowerUnpegged")
	}
	if (l.Flags>>Secret)&1 == 1 {
		data = append(data, "secret")
	}
	if (l.Flags>>BlockSound)&1 == 1 {
		data = append(data, "blockSound")
	}
	if (l.Flags>>NotOnMap)&1 == 1 {
		data = append(data, "notOnMap")
	}
	if (l.Flags>>AlreadyOnMap)&1 == 1 {
		data = append(data, "alreadyOnMap")
	}
	return strings.Join(data, ",")
}

// NewLineDefs reads LineDef data from the file based on lumpInfo and returns a slice of LineDef pointers or an error.
func NewLineDefs(f *os.File, lumpInfo *LumpInfo) ([]*LineDef, error) {
	var pLineDef LineDef
	count := int(lumpInfo.Size) / int(unsafe.Sizeof(pLineDef))
	pLineDefs := make([]LineDef, count, count)
	if err := binary.Read(f, binary.LittleEndian, pLineDefs); err != nil {
		return nil, err
	}
	lineDefs := make([]*LineDef, count, count)
	for idx, ld := range pLineDefs {
		lineDefs[idx] = &LineDef{
			VertexStart:  ld.VertexStart,
			VertexEnd:    ld.VertexEnd,
			Flags:        ld.Flags,
			Function:     ld.Function,
			Tag:          ld.Tag,
			SideDefRight: ld.SideDefRight,
			SideDefLeft:  ld.SideDefLeft,
		}
	}
	return lineDefs, nil
}
