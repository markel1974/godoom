package lumps

import (
	"encoding/binary"
	"os"
	"unsafe"
)

// TexInfo represents texture mapping information, including texture vectors, texture index, and special flags.
type TexInfo struct {
	Vecs   [2][4]float32 // Vecs[0] = Asse S (X,Y,Z, Offset), Vecs[1] = Asse T (X,Y,Z, Offset)
	MipTex uint32        // Indice della texture nel lump TEXTURES
	Flags  uint32        // Flag speciali (es. acqua, cielo)
}

// NewTexInfos reads texture information from the provided file based on lump metadata and returns a slice of TexInfo pointers.
func NewTexInfos(f *os.File, lumpInfo *LumpInfo) ([]*TexInfo, error) {
	var pTexInfo TexInfo
	count := int(lumpInfo.Size) / int(unsafe.Sizeof(pTexInfo))
	pTexInfos := make([]TexInfo, count)
	if err := binary.Read(f, binary.LittleEndian, pTexInfos); err != nil {
		return nil, err
	}
	texInfos := make([]*TexInfo, count)
	for idx, t := range pTexInfos {
		texInfos[idx] = &TexInfo{
			Vecs:   t.Vecs,
			MipTex: t.MipTex,
			Flags:  t.Flags,
		}
	}
	return texInfos, nil
}
