package lumps

import (
	"io"
)

// NewEntities3 reads and parses the entities lump from a Quake 3 BSP file into a slice of Entity pointers.
func NewEntities3(rs io.ReadSeeker, header Headers3) ([]*Entity, error) {
	lump := header.Lumps[LumpEntities3]
	if _, err := rs.Seek(int64(lump.Offset), io.SeekStart); err != nil {
		return nil, err
	}
	data := make([]byte, lump.Length)
	if _, err := rs.Read(data); err != nil {
		return nil, err
	}
	return NewEntitiesFromText(FromNullTerminatingString(data))
}
