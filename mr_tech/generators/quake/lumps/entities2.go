package lumps

import "io"

// NewEntities2 reads and parses entity data from a file using a read-seeker and header metadata and returns a slice of entities.
func NewEntities2(rs io.ReadSeeker, headers Headers2) ([]*Entity, error) {
	lump := headers.Lumps[LumpEntities2]
	if _, err := rs.Seek(int64(lump.Offset), io.SeekStart); err != nil {
		return nil, err
	}
	data := make([]byte, lump.Length)
	if _, err := rs.Read(data); err != nil {
		return nil, err
	}
	text := FromNullTerminatingString(data)
	return NewEntitiesFromText(text)
}
