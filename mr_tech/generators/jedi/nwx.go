package jedi

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"unsafe"
)

// NWXCell represents a single cell in the NWX structure with metadata and pixel data.
type NWXCell struct {
	id     string
	offset int64
	sizeX  int
	sizeY  int
	pixels []byte
}

// NewNWXCell creates a new NWXCell instance with the given ID and offset.
func NewNWXCell(id string, offset int64) *NWXCell {
	return &NWXCell{
		id:     id,
		offset: offset,
	}
}

// GetId returns the unique identifier of the NWXCell instance.
func (p *NWXCell) GetId() string { return p.id }

// GetSize returns the width and height of the NWXCell as two integers.
func (p *NWXCell) GetSize() (int, int) { return p.sizeX, p.sizeY }

// GetPixels returns the raw pixel data of the NWXCell as a byte slice.
func (p *NWXCell) GetPixels() []byte { return p.pixels }

// Parse reads NWX cell data from the provided io.ReadSeeker, parsing pixel data and initializing internal structures.
func (p *NWXCell) Parse(r io.ReadSeeker, streamW, streamH, streamSize int) error {
	if _, err := r.Seek(p.offset, io.SeekStart); err != nil {
		return err
	}

	p.sizeX, p.sizeY = streamW, streamH
	p.pixels = make([]byte, streamW*streamH)

	colTable := make([]uint32, streamW)
	if err := binary.Read(r, binary.LittleEndian, &colTable); err != nil {
		return err
	}

	for x := 0; x < streamW; x++ {
		target := colTable[x] & 0x00FFFFFF
		if target == 0 {
			continue
		}

		// CALCOLO CHUNK SIZE DELLA COLONNA (come da offset table C++)
		var nextTarget = int64(streamSize)
		for nextX := x + 1; nextX < streamW; nextX++ {
			nt := colTable[nextX] & 0x00FFFFFF
			if nt != 0 {
				nextTarget = int64(nt)
				break
			}
		}
		maxColBytes := int(nextTarget - int64(target))

		currentOffset := p.offset + int64(target)
		if _, err := r.Seek(currentOffset, io.SeekStart); err != nil {
			return err
		}

		bytesRead := 0
		y := 0

		// Il limite ora è maxColBytes, non l'intero streamSize
		for bytesRead < maxColBytes && y < streamH {
			var controlByte uint8
			if err := binary.Read(r, binary.LittleEndian, &controlByte); err != nil {
				return err
			}
			bytesRead++

			count := int(controlByte>>1) + 1
			isRLE := (controlByte & 1) == 1

			if isRLE {
				var color uint8
				if err := binary.Read(r, binary.LittleEndian, &color); err != nil {
					return err
				}
				bytesRead++
				for i := 0; i < count && y < streamH; i++ {
					drawY := (streamH - 1) - y
					targetIdx := (drawY * streamW) + x
					if targetIdx >= 0 && targetIdx < len(p.pixels) {
						p.pixels[targetIdx] = color
					}
					y++
				}
			} else {
				for i := 0; i < count && y < streamH; i++ {
					var color uint8
					if err := binary.Read(r, binary.LittleEndian, &color); err != nil {
						return err
					}
					bytesRead++
					drawY := (streamH - 1) - y
					targetIdx := (drawY * streamW) + x
					if targetIdx >= 0 && targetIdx < len(p.pixels) {
						p.pixels[targetIdx] = color
					}
					y++
				}
			}
		}
	}
	return nil
}

// NWXCellHeader represents metadata for a specific cell structure in a binary NWX file format.
// physIndex denotes the physical index of the cell.
// size defines the size of the cell in bytes.
// streamW specifies the width of the cell's data stream.
// streamH specifies the height of the cell's data stream.
type NWXCellHeader struct {
	physIndex uint32
	size      uint32
	streamW   uint32
	streamH   uint32
}

// NewNWXCellHeader creates and returns a new instance of NWXCellHeader with default, uninitialized values.
func NewNWXCellHeader() *NWXCellHeader {
	return &NWXCellHeader{}
}

// Parse reads binary data from the given io.ReadSeeker and populates the NWXCellHeader fields. Returns an error on failure.
func (h *NWXCellHeader) Parse(r io.ReadSeeker) error {
	var raw struct {
		PhysIndex uint32
		Size      uint32
		StreamW   uint32
		StreamH   uint32
		Magic     uint32
	}
	if err := binary.Read(r, binary.LittleEndian, &raw); err != nil {
		return err
	}
	h.physIndex = raw.PhysIndex
	h.size = raw.Size
	h.streamW = raw.StreamW
	h.streamH = raw.StreamH
	return nil
}

// NWXFrameHeader represents the header structure for an NWX frame with an offset field indicating its position in a file.
type NWXFrameHeader struct {
	offset int64
}

// NewNWXFrameHeader creates a new NWXFrameHeader instance with the specified offset.
func NewNWXFrameHeader(offset int64) *NWXFrameHeader {
	return &NWXFrameHeader{
		offset: offset,
	}
}

// Parse reads NWX frame header data from the provided io.ReadSeeker and populates the structure.
func (h *NWXFrameHeader) Parse(r io.ReadSeeker) error {
	// FrmtHeader must be 32byte
	var frmtHeader struct {
		Magic     uint32
		Tag       uint32
		Flags     uint32
		InsertX   uint32
		InsertY   uint32
		Flip      uint32
		Width     float32
		Height    float32
		CellIndex uint32
		Pad7      uint32
	}
	if _, err := r.Seek(h.offset, io.SeekStart); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &frmtHeader); err != nil {
		return err
	}

	return nil
}

// NWXCeltHeader represents the header structure for a CELT file, containing offset, cell count, and chunk size.
type NWXCeltHeader struct {
	offset    int64
	numCells  uint32
	chunkSize uint32
}

// NewNWXCeltHeader creates and returns a new instance of NWXCeltHeader with the specified file offset.
func NewNWXCeltHeader(offset int64) *NWXCeltHeader {
	return &NWXCeltHeader{
		offset: offset,
	}
}

// Parse reads and validates the CELT header from the provided io.ReadSeeker and updates the NWXCeltHeader fields.
func (h *NWXCeltHeader) Parse(r io.ReadSeeker) error {
	var celtHeader struct {
		Magic     uint32
		NumCells  uint32
		ChunkSize uint32
	}
	if _, err := r.Seek(h.offset, io.SeekStart); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &celtHeader); err != nil {
		return err
	}
	if celtHeader.Magic != 1414284611 { // "CELT"
		return fmt.Errorf("invalid CELT header magic")
	}
	h.numCells = celtHeader.NumCells
	h.chunkSize = celtHeader.ChunkSize
	return nil
}

// NWXFrame represents a frame in the NWX format, storing metadata and a reference to an associated NWXCell.
type NWXFrame struct {
	CellIndex     uint32
	PhysicalIndex uint32
	Width         float32
	Height        float32
	Cell          *NWXCell
}

// NewNWXFrame creates and returns a new instance of NWXFrame with default values.
func NewNWXFrame() *NWXFrame {
	return &NWXFrame{}
}

// Parse reads binary data from the provided io.ReadSeeker and populates the NWXFrame fields. It returns the size of parsed data.
func (g *NWXFrame) Parse(r io.ReadSeeker) (int, error) {
	var raw struct {
		InsertX       int32  // Offset X per centrare l'immagine
		InsertY       int32  // Offset Y per centrare l'immagine
		Flags         uint32 // Flip orizzontale/verticale o attributi
		CellIndex     uint32 // ID della cella nel blocco CELT (i pixel reali)
		Pad1          uint32
		Pad2          uint32
		Width         float32 // Dimensione Float (Fixed Point tradotto) o Bounding Box
		Height        float32 // Dimensione Float
		Pad3          uint32
		PhysicalIndex uint32
	}
	if err := binary.Read(r, binary.LittleEndian, &raw); err != nil {
		return 0, err
	}
	g.CellIndex = raw.CellIndex
	g.PhysicalIndex = raw.PhysicalIndex
	g.Width = raw.Width
	g.Height = raw.Height
	g.Cell = nil
	v := int(unsafe.Sizeof(raw))
	return v, nil
}

type NWXSeqNode struct {
	marker int16
	index  int16
	tick   uint32
	pad    uint32
}

type NWXAction struct {
	id    int
	size  uint32
	nodes []*NWXSeqNode
}

// NWXSequencer represents the header metadata for a sequence in an NWX file format.
type NWXSequencer struct {
	offset       int64
	chunkSize    uint32
	numSequences uint32 // Es: 32 azioni
	actions      []*NWXAction
}

func NewNWXSequencer(offset int64) *NWXSequencer {
	return &NWXSequencer{
		offset: offset,
	}
}

func (s *NWXSequencer) Parse(r io.ReadSeeker) error {
	var raw struct {
		Magic        uint32 // "SEQT"
		NumSequences uint32
		ChunkSize    uint32
	}

	if _, err := r.Seek(s.offset, io.SeekStart); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &raw); err != nil {
		return err
	}
	if raw.Magic != 1414481987 {
		return fmt.Errorf("invalid SEQT sequencer magic")
	}

	s.numSequences = raw.NumSequences
	s.chunkSize = raw.ChunkSize
	s.actions = make([]*NWXAction, 0, s.numSequences)

	for i := uint32(0); i < s.numSequences; i++ {
		// 1. Leggiamo il Tag iniziale del blocco (esterno al conteggio della Size)
		var blockTag uint32
		if err := binary.Read(r, binary.LittleEndian, &blockTag); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return err
		}
		// 2. Salviamo l'offset esatto da cui la Size inizia a contare
		chunkStart, err := r.Seek(0, io.SeekCurrent)
		if err != nil {
			return err
		}

		// 3. Leggiamo l'intero header in un colpo solo
		var header struct {
			Size      uint32
			Hex1      uint32
			Hex2      uint32
			Int1      uint32
			Int2      uint32
			NodeCount uint32
			Int4      uint32
		}
		if err = binary.Read(r, binary.LittleEndian, &header); err != nil {
			return err
		}

		action := &NWXAction{
			id:   int(i),
			size: header.Size,
		}

		// 4. Leggiamo il payload dei nodi
		if header.NodeCount > 0 {
			type rawSeqNode struct {
				Flags uint32
				Tick  uint32
				Pad   uint32
			}
			nodes := make([]rawSeqNode, header.NodeCount)
			if err = binary.Read(r, binary.LittleEndian, nodes); err != nil {
				if err == io.EOF || errors.Is(err, io.ErrUnexpectedEOF) {
					break
				}
				return err
			}

			action.nodes = make([]*NWXSeqNode, header.NodeCount)
			for x := 0; x < len(nodes); x++ {
				action.nodes[x] = &NWXSeqNode{
					marker: int16(nodes[x].Flags >> 16),
					index:  int16(nodes[x].Flags & 0xFFFF),
					tick:   nodes[x].Tick,
					pad:    nodes[x].Pad,
				}
			}
		}
		s.actions = append(s.actions, action)

		// Riallineamento
		chunkNext := chunkStart + int64(header.Size)
		if _, err = r.Seek(chunkNext, io.SeekStart); err != nil {
			return err
		}
	}

	return nil
}

// NWXHeader represents the header structure of an NWX file format, containing necessary offsets for parsing data.
type NWXHeader struct {
	offset     int64
	celtOffset uint32
	frmtOffset uint32
	seqOffset  uint32
}

// NewNWXHeader creates and returns a new NWXHeader instance with the specified offset.
func NewNWXHeader(offset int64) *NWXHeader {
	return &NWXHeader{
		offset: offset,
	}
}

// Parse reads and validates the NWX header from the provided io.ReadSeeker, extracting offsets for CELT, FRMT, and SEQ sections.
func (h *NWXHeader) Parse(r io.ReadSeeker) error {
	var raw struct {
		Signature  [4]byte
		Version    uint32
		Unknown    uint32
		ScaleX     float32
		ScaleY     float32
		CeltOffset uint32
		FrmtOffset uint32
		SeqOffset  uint32
	}
	if _, err := r.Seek(h.offset, io.SeekStart); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &raw); err != nil {
		return err
	}
	if string(raw.Signature[:]) != "WAXF" {
		return fmt.Errorf("invalid signature")
	}
	h.celtOffset = raw.CeltOffset
	h.frmtOffset = raw.FrmtOffset
	h.seqOffset = raw.SeqOffset
	return nil
}

// NWX represents a collection of NWXFrame instances used for handling graphical frames in the NWX format.
type NWX struct {
	//actions []*NWXSequence
	frames []*NWXFrame
}

// NewNWX initializes and returns a pointer to a new NWX instance.
func NewNWX() *NWX {
	return &NWX{}
}

// Parse reads, decodes, and processes NWX data from the provided io.ReadSeeker, initializing internal frames and cell structures.
func (w *NWX) Parse(baseId string, r io.ReadSeeker) error {
	waxHeader := NewNWXHeader(0)
	if err := waxHeader.Parse(r); err != nil {
		return err
	}

	seqHeader := NewNWXSequencer(int64(waxHeader.seqOffset))
	if err := seqHeader.Parse(r); err != nil {
		return err
	}

	//w.frames = make([]*NWXFrame, seqHeader.numFrames+1)
	frmtHeader := NewNWXFrameHeader(int64(waxHeader.frmtOffset))
	if err := frmtHeader.Parse(r); err != nil {
		return err
	}

	celtHeader := NewNWXCeltHeader(int64(waxHeader.celtOffset))
	if err := celtHeader.Parse(r); err != nil {
		return err
	}

	cellCache := make(map[uint32]*NWXCell)

	for i := uint32(0); i < celtHeader.numCells; i++ {
		cellHeader := NewNWXCellHeader()
		if err := cellHeader.Parse(r); err != nil {
			return err
		}
		//TODO REAL IMPLEMENTATION
		f := NewNWXFrame()
		f.CellIndex = cellHeader.physIndex
		f.PhysicalIndex = cellHeader.physIndex
		f.Width = float32(cellHeader.streamW)
		f.Height = float32(cellHeader.streamH)
		w.frames = append(w.frames, f)

		// Ora siamo posizionati PERFETTAMENTE all'inizio della colTable (payload vero)
		payloadStart, _ := r.Seek(0, io.SeekCurrent)

		if cell := cellCache[cellHeader.physIndex]; cell != nil {
			f.Cell = cellCache[cellHeader.physIndex]
		} else {
			cellId := fmt.Sprintf("phys_%d", cellHeader.physIndex)
			f.Cell = NewNWXCell(cellId, payloadStart)
			if cellHeader.streamW > 0 && cellHeader.streamH > 0 {
				if err := f.Cell.Parse(r, int(cellHeader.streamW), int(cellHeader.streamH), int(cellHeader.size)); err != nil {
					return err
				}
			}
			cellCache[cellHeader.physIndex] = f.Cell
		}
		if _, err := r.Seek(payloadStart+int64(cellHeader.size), io.SeekStart); err != nil {
			return err
		}
	}
	return nil
}
