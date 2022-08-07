package wad

import (
	"encoding/binary"
	"fmt"
	"os"
	"sort"
	"unsafe"
)

type LumpInfo struct {
	Filepos int64
	Size    int32
	Name    string
}

type Texture struct {
	Header  *TextureHeader
	Patches []*Patch
}

type TextureHeader struct {
	TexName         string
	Masked          int32
	Width           int16
	Height          int16
	ColumnDirectory int32
	NumPatches      int16
}

type Patch struct {
	XOffset     int16
	YOffset     int16
	PNameNumber int16
	StepDir     int16
	ColorMap    int16
}

type Image struct {
	Width  int
	Height int
	Pixels []byte
}

type PictureHeader struct {
	Width      int16
	Height     int16
	LeftOffset int16
	TopOffset  int16
}

type Flat struct {
	Data []byte
}

type Level struct {
	Things     []*Thing
	LineDefs   []*LineDef
	SideDefs   []*SideDef
	Vertexes   []*Vertex
	Segments   []*Seg
	SubSectors []*SubSector
	Nodes      []*Node
	Sectors    []*Sector
}

type Thing struct {
	XPosition int16
	YPosition int16
	Angle     int16
	Type      int16
	Options   int16
}

type LineDef struct {
	VertexStart  int16
	VertexEnd    int16
	Flags        int16
	Function     int16
	Tag          int16
	SideDefRight int16
	SideDefLeft  int16
}

type SideDef struct {
	XOffset       int16
	YOffset       int16
	UpperTexture  string
	LowerTexture  string
	MiddleTexture string
	SectorRef     int16
}

type Vertex struct {
	XCoord int16
	YCoord int16
}

type Seg struct {
	VertexStart   int16
	VertexEnd     int16
	Bams          int16
	LineNum       int16
	SegmentSide   int16
	SegmentOffset int16
}

type SubSector struct {
	NumSegments int16
	StartSeg    int16
}

type BBox struct {
	Top    int16
	Bottom int16
	Left   int16
	Right  int16
}

type Node struct {
	X     int16
	Y     int16
	DX    int16
	DY    int16
	BBox  [2]BBox
	Child [2]int16
}

type Sector struct {
	FloorHeight   int16
	CeilingHeight int16
	FloorPic      string
	CeilingPic    string
	LightLevel    int16
	SpecialSector int16
	Tag           int16
}

type Reject struct {
}

type BlockMap struct {
}

type RGB struct {
	Red   uint8
	Green uint8
	Blue  uint8
}

type Palette struct {
	Table [256]RGB
}

type PlayPal struct {
	Palettes [14]Palette
}



type WAD struct {
	pNames                  []string
	patches                 map[string]*Image
	playPal                 *PlayPal
	textures                map[string]*Texture
	flats                   map[string]*Flat
	levels                  map[string]int
	lumps                   map[string]int
	lumpInfos               []*LumpInfo
	file                    *os.File
	transparentPaletteIndex byte
}



func New() *WAD {
	return &WAD{
		transparentPaletteIndex: 255,
	}
}

func (w * WAD) Load(filename string) error {
	var err error
	if w.file, err = os.Open(filename); err != nil { return err }
	l := NewLoader()
	return l.Setup(w)
}

func (w *WAD) GetTexture(name string) (*Texture, bool) {
	texture, ok := w.textures[name]
	return texture, ok
}

func (w *WAD) GetImage(pNameNumber int16) (*Image, bool) {
	image, ok := w.patches[w.pNames[pNameNumber]]
	return image, ok
}

func (w *WAD) GetFlat(flatName string) (*Flat, bool) {
	flat, ok := w.flats[flatName]
	return flat, ok
}

func (w *WAD) GetLevels() []string {
	var result []string
	for name := range w.levels {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func (w *WAD) GetLevel(levelName string) (*Level, error) {
	var err error
	level := &Level{}
	levelIdx := w.levels[levelName]
	for i := levelIdx + 1; i < levelIdx+11; i++ {
		lumpInfo := w.lumpInfos[i]
		if err := Seek(w.file, lumpInfo.Filepos); err != nil { return nil, err }
		switch lumpInfo.Name {
		case "THINGS": if level.Things, err = w.readThings(lumpInfo); err != nil { return nil, err }
		case "SIDEDEFS": if level.SideDefs, err = w.readSideDefs(lumpInfo); err != nil { return nil, err }
		case "LINEDEFS": if level.LineDefs, err = w.readLineDefs(lumpInfo); err != nil { return nil, err }
		case "VERTEXES": if level.Vertexes, err = w.readVertexes(lumpInfo); err != nil { return nil, err }
		case "SEGS": if level.Segments, err = w.readSegments(lumpInfo); err != nil { return nil, err }
		case "SSECTORS": if level.SubSectors, err = w.readSubSectors(lumpInfo); err != nil { return nil, err }
		case "NODES": if level.Nodes, err = w.readNodes(lumpInfo); err != nil { return nil, err }
		case "SECTORS": if level.Sectors, err = w.readSectors(lumpInfo); err != nil { return nil, err }
		default: fmt.Printf("Unhandled lump %s\n", lumpInfo.Name)
		}
	}
	return level, nil
}

func (w *WAD) readThings(lumpInfo *LumpInfo) ([]*Thing, error) {
	var pThing Thing
	count := int(lumpInfo.Size) / int(unsafe.Sizeof(pThing))
	pThings := make([]Thing, count, count)
	if err := binary.Read(w.file, binary.LittleEndian, pThings); err != nil {
		return nil, err
	}
	things := make([]*Thing, count, count)
	for idx, t := range pThings {
		things[idx] = &Thing{
			XPosition: t.XPosition,
			YPosition: t.YPosition,
			Angle:     t.Angle,
			Type:      t.Type,
			Options:   t.Options,
		}
	}
	return things, nil
}

func (w *WAD) readLineDefs(lumpInfo *LumpInfo) ([]*LineDef, error) {
	var pLineDef LineDef
	count := int(lumpInfo.Size) / int(unsafe.Sizeof(pLineDef))
	pLineDefs := make([]LineDef, count, count)
	if err := binary.Read(w.file, binary.LittleEndian, pLineDefs); err != nil {
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

func (w *WAD) readSideDefs(lumpInfo *LumpInfo) ([]*SideDef, error) {
	type PrivateSideDef struct {
		XOffset       int16
		YOffset       int16
		UpperTexture  [8]byte
		LowerTexture  [8]byte
		MiddleTexture [8]byte
		SectorRef     int16
	}
	var pSideDef PrivateSideDef
	count := int(lumpInfo.Size) / int(unsafe.Sizeof(pSideDef))
	pSideDefs := make([]PrivateSideDef, count, count)
	if err := binary.Read(w.file, binary.LittleEndian, pSideDefs); err != nil {
		return nil, err
	}
	sideDef := make([]*SideDef, count, count)
	for idx, p := range pSideDefs {
		sideDef[idx] = &SideDef{
			XOffset:       p.XOffset,
			YOffset:       p.YOffset,
			UpperTexture:  ToString(p.UpperTexture),
			LowerTexture:  ToString(p.LowerTexture),
			MiddleTexture: ToString(p.MiddleTexture),
			SectorRef:     p.SectorRef,
		}
	}
	return sideDef, nil
}

func (w *WAD) readVertexes(lumpInfo *LumpInfo) ([]*Vertex, error) {
	var pVertex Vertex
	count := int(lumpInfo.Size) / int(unsafe.Sizeof(pVertex))
	pVertexes := make([]Vertex, count, count)
	if err := binary.Read(w.file, binary.LittleEndian, pVertexes); err != nil {
		return nil, err
	}
	vertexes := make([]*Vertex, count, count)
	for idx, v := range pVertexes {
		vertexes[idx] = &Vertex{
			XCoord: v.XCoord,
			YCoord: v.YCoord,
		}
	}
	return vertexes, nil
}

func (w *WAD) readSegments(lumpInfo *LumpInfo) ([]*Seg, error) {
	var pSeg Seg
	count := int(lumpInfo.Size) / int(unsafe.Sizeof(pSeg))
	pSegments := make([]Seg, count, count)
	if err := binary.Read(w.file, binary.LittleEndian, pSegments); err != nil {
		return nil, err
	}
	segments := make([]*Seg, count, count)
	for idx, s := range pSegments {
		segments[idx] = &Seg{
			VertexStart:   s.VertexStart,
			VertexEnd:     s.VertexEnd,
			Bams:          s.Bams,
			LineNum:       s.LineNum,
			SegmentSide:   s.SegmentSide,
			SegmentOffset: s.SegmentOffset,
		}
	}
	return segments, nil
}

func (w *WAD) readSubSectors(lumpInfo *LumpInfo) ([]*SubSector, error) {
	var pSubSector SubSector
	count := int(lumpInfo.Size) / int(unsafe.Sizeof(pSubSector))
	pSubSectors := make([]SubSector, count, count)
	if err := binary.Read(w.file, binary.LittleEndian, pSubSectors); err != nil {
		return nil, err
	}
	subSectors := make([]*SubSector, count, count)
	for idx, s := range pSubSectors {
		subSectors[idx] = &SubSector{
			NumSegments: s.NumSegments,
			StartSeg:    s.StartSeg,
		}
	}
	return subSectors, nil
}

func (w *WAD) readNodes(lumpInfo *LumpInfo) ([]*Node, error) {
	var pNode Node
	count := int(lumpInfo.Size) / int(unsafe.Sizeof(pNode))
	pNodes := make([]Node, count, count)
	if err := binary.Read(w.file, binary.LittleEndian, pNodes); err != nil {
		return nil, err
	}
	nodes := make([]*Node, count, count)
	for idx, n := range pNodes {
		nodes[idx] = &Node{
			X:     n.X,
			Y:     n.Y,
			DX:    n.DX,
			DY:    n.DY,
			BBox:  n.BBox,
			Child: n.Child,
		}
	}
	return nodes, nil
}

func (w *WAD) readSectors(lumpInfo *LumpInfo) ([]*Sector, error) {
	type privateSector struct {
		FloorHeight   int16
		CeilingHeight int16
		FloorPic      [8]byte
		CeilingPic    [8]byte
		LightLevel    int16
		SpecialSector int16
		Tag           int16
	}
	var pSector privateSector
	count := int(lumpInfo.Size) / int(unsafe.Sizeof(pSector))
	pSectors := make([]privateSector, count, count)
	if err := binary.Read(w.file, binary.LittleEndian, pSectors); err != nil {
		return nil, err
	}
	sectors := make([]*Sector, count, count)
	for idx, p := range pSectors {
		sectors[idx] = &Sector{
			FloorHeight:   p.FloorHeight,
			CeilingHeight: p.CeilingHeight,
			FloorPic:      ToString(p.FloorPic),
			CeilingPic:    ToString(p.CeilingPic),
			LightLevel:    p.LightLevel,
			SpecialSector: p.SpecialSector,
			Tag:           p.Tag,
		}
	}
	return sectors, nil
}


