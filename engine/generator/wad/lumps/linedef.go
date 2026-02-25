package lumps

import (
	"encoding/binary"
	"os"
	"strings"
	"unsafe"
)

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

type LineDef struct {
	VertexStart  int16
	VertexEnd    int16
	Flags        int16
	Function     int16
	Tag          int16
	SideDefRight int16
	SideDefLeft  int16
}

func (l *LineDef) HasFlag(flag int) bool {
	return l.Flags>>flag&1 == 1
}

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
