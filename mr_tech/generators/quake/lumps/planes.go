package lumps

import (
	"encoding/binary"
	"os"
	"unsafe"
)

// Plane represents a 3D plane defined by a normal vector, a distance from the origin, and a type indicator.
type Plane struct {
	Normal [3]float32
	Dist   float32
	Type   int32 // 0-2 assiali (X,Y,Z), 3-5 non assiali
}

// NewPlanes reads and parses plane data from the given file based on the provided lump information.
func NewPlanes(f *os.File, lumpInfo *LumpInfo) ([]*Plane, error) {
	var pPlane Plane
	count := int(lumpInfo.Size) / int(unsafe.Sizeof(pPlane))
	pPlanes := make([]Plane, count)

	if err := binary.Read(f, binary.LittleEndian, pPlanes); err != nil {
		return nil, err
	}

	planes := make([]*Plane, count)
	for idx, p := range pPlanes {
		planes[idx] = &Plane{
			Normal: p.Normal,
			Dist:   p.Dist,
			Type:   p.Type,
		}
	}
	return planes, nil
}
