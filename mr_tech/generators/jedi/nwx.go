package jedi

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// NWXCell represents a cell structure holding pixel data for graphical rendering.
type NWXCell struct {
	offset int64
	sizeX  int
	sizeY  int
	pixels []byte
}

// NewNWXCell creates and initializes a new NWXCell instance with the provided id and offset.
func NewNWXCell(offset int64) *NWXCell {
	return &NWXCell{
		offset: offset,
	}
}

// GetSize returns the dimensions of the NWXCell as two integers: width (sizeX) and height (sizeY).
func (p *NWXCell) GetSize() (int, int) { return p.sizeX, p.sizeY }

// GetPixels returns the raw pixel data of the NWXCell as a slice of bytes.
func (p *NWXCell) GetPixels() []byte { return p.pixels }

// Parse reads pixel data from the provided stream using offset and size parameters and processes it into columns and rows.
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
					targetIdx := (y * streamW) + x
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
					targetIdx := (y * streamW) + x
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

// NWXCellHeader represents metadata for a cell in the NWX format.
// index specifies the physical index of the cell.
// size denotes the size of the cell in bytes.
// streamW indicates the width of the cell stream.
// streamH indicates the height of the cell stream.
type NWXCellHeader struct {
	index   uint32
	size    uint32
	streamW uint32
	streamH uint32
}

// NewNWXCellHeader creates and initializes a new instance of NWXCellHeader with default values.
func NewNWXCellHeader() *NWXCellHeader {
	return &NWXCellHeader{}
}

// Parse decodes NWXCellHeader data from the provided io.ReadSeeker and initializes its fields.
func (h *NWXCellHeader) Parse(r io.ReadSeeker) error {
	var raw struct {
		Index   uint32
		Size    uint32
		StreamW uint32
		StreamH uint32
		Magic   uint32
	}
	if err := binary.Read(r, binary.LittleEndian, &raw); err != nil {
		return err
	}
	h.index = raw.Index
	h.size = raw.Size
	h.streamW = raw.StreamW
	h.streamH = raw.StreamH
	return nil
}

// NWXVerbFRMT represents a structure holding an offset and a collection of NWXFrame elements used in rendering.
type NWXVerbFRMT struct {
	offset int64
	frames []*NWXFrame
}

// NewNWXVerbFRMT creates and returns a new NWXVerbFRMT instance with the specified offset for parsing FRMT sections.
func NewNWXVerbFRMT(offset int64) *NWXVerbFRMT {
	return &NWXVerbFRMT{
		offset: offset,
	}
}

// Parse reads and processes the FRMT section of an NWX file, initializing frames and verifying the format's validity.
func (f *NWXVerbFRMT) Parse(r io.ReadSeeker) error {
	var raw struct {
		Magic     uint32
		NumFrames uint32
		ChunkSize uint32
	}
	if _, err := r.Seek(f.offset, io.SeekStart); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &raw); err != nil {
		return err
	}
	if raw.Magic != 1414353478 { // "FRMT"
		return fmt.Errorf("invalid FRMT format magic")
	}
	if _, err := r.Seek(0, io.SeekCurrent); err != nil {
		return err
	}
	f.frames = make([]*NWXFrame, raw.NumFrames)
	for i := uint32(0); i < raw.NumFrames; i++ {
		frame := NewNWXFrame()
		if err := frame.Parse(r); err != nil {
			return err
		}
		f.frames[i] = frame
	}
	return nil
}

// NWXVerbCELT represents metadata for CELT chunk processing, including offset, number of cells, and chunk size.
type NWXVerbCELT struct {
	offset    int64
	numCells  uint32
	chunkSize uint32
}

// NewNWXVerbCELT creates and returns a new NWXVerbCELT with the specified offset.
func NewNWXVerbCELT(offset int64) *NWXVerbCELT {
	return &NWXVerbCELT{
		offset: offset,
	}
}

// Parse reads and interprets the CELT header from the provided io.ReadSeeker and populates the NWXCeltHeader fields.
func (h *NWXVerbCELT) Parse(r io.ReadSeeker) error {
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

// NWXFrame represents a frame in an NWX structure, associating rendering dimensions and cell metadata.
type NWXFrame struct {
	cellIndex uint32
	width     float32
	height    float32
	cell      *NWXCell
}

// NewNWXFrame creates and returns a new instance of NWXGraphicalFrame with default values.
func NewNWXFrame() *NWXFrame {
	return &NWXFrame{}
}

// Parse reads and extracts graphical frame data from the provided io.ReadSeeker stream. It returns the size of the parsed data and an error if any occurs.
func (g *NWXFrame) Parse(r io.ReadSeeker) error {
	var raw struct {
		Unknown1      [12]byte // Byte 0-11 (Header/Flags vuoti)
		Width         float32  // Byte 12-15 (Valore: 55.0)
		Height        float32  // Byte 16-19 (Valore: 98.0)
		CellIndex     uint32   // Byte 20-23 (Valore: 1)
		Pad           uint32   // Byte 24-27
		InsertX       int32    // Byte 28-31 (Valore: -31)
		InsertY       int32    // Byte 32-35 (Valore: -90)
		PhysicalIndex uint32   // Byte 36-39 (Valore: 16)
	}
	if err := binary.Read(r, binary.LittleEndian, &raw); err != nil {
		return err
	}
	g.cellIndex = raw.CellIndex
	g.width = raw.Width
	g.height = raw.Height
	g.cell = nil
	return nil
}

// NWXSeqNode represents a node in a sequence, storing metadata about its state and associated cell.
type NWXSeqNode struct {
	marker int16
	index  int16
	tick   uint32
	pad    uint32
	id     string
	cell   *NWXCell
}

// GetId returns the unique identifier of the NWXSeqNode as a string.
func (n *NWXSeqNode) GetId() string {
	return n.id
}

// GetCell retrieves the associated NWXCell of the NWXSeqNode, which contains graphical rendering data.
func (n *NWXSeqNode) GetCell() *NWXCell {
	return n.cell
}

// NWXAction represents an action containing a unique identifier, a size, and a slice of sequence node pointers.
type NWXAction struct {
	id    int
	size  uint32
	nodes []*NWXSeqNode
}

// GetNodes returns the slice of NWXSeqNode pointers associated with the NWXAction.
func (n *NWXAction) GetNodes() []*NWXSeqNode {
	return n.nodes
}

// NWXVerbSEQT is a structure representing a sequencer with associated actions and metadata for processing sequences.
type NWXVerbSEQT struct {
	offset       int64
	chunkSize    uint32
	numSequences uint32
	actions      []*NWXAction
}

// NewNWXVerbSEQT creates a new instance of NWXVerbSEQT with the specified offset value.
func NewNWXVerbSEQT(offset int64) *NWXVerbSEQT {
	return &NWXVerbSEQT{
		offset: offset,
	}
}

// Parse reads and parses sequencer data from the given io.ReadSeeker, initializing sequences and actions for NWXVerbSEQT.
func (s *NWXVerbSEQT) Parse(r io.ReadSeeker) error {
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
		// Leggiamo il Tag iniziale del blocco (esterno al conteggio della Size)
		var blockTag uint32
		if err := binary.Read(r, binary.LittleEndian, &blockTag); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return err
		}
		chunkStart, err := r.Seek(0, io.SeekCurrent)
		if err != nil {
			return err
		}

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

// NWXHeader represents metadata offsets required for parsing a NWX file.
// offset specifies the starting byte position for the header in the file.
// celtOffset is the byte offset for the CELT section.
// frmtOffset is the byte offset for the FRMT section.
// seqOffset is the byte offset for the SEQ section.
type NWXHeader struct {
	offset     int64
	celtOffset uint32
	frmtOffset uint32
	seqOffset  uint32
}

// NewNWXHeader creates a new NWXHeader instance with the specified offset.
func NewNWXHeader(offset int64) *NWXHeader {
	return &NWXHeader{
		offset: offset,
	}
}

// Parse reads and processes NWX header data from the provided io.ReadSeeker, updating the struct fields with parsed offsets.
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

// NWX represents a container structure that holds frames and a sequencer for managing NWX data.
type NWX struct {
	sequencer *NWXVerbSEQT
}

// NewNWX creates and returns a new instance of the NWX structure.
func NewNWX() *NWX {
	return &NWX{}
}

// Parse processes the NWX file data from the provided reader and initializes NWXHeader, NWXVerbSEQT, and related components.
func (w *NWX) Parse(baseId string, r io.ReadSeeker) error {
	waxHeader := NewNWXHeader(0)
	if err := waxHeader.Parse(r); err != nil {
		return err
	}

	w.sequencer = NewNWXVerbSEQT(int64(waxHeader.seqOffset))
	if err := w.sequencer.Parse(r); err != nil {
		return err
	}

	frmt := NewNWXVerbFRMT(int64(waxHeader.frmtOffset))
	if err := frmt.Parse(r); err != nil {
		return err
	}

	celt := NewNWXVerbCELT(int64(waxHeader.celtOffset))
	if err := celt.Parse(r); err != nil {
		return err
	}

	cellCache := make(map[int16]*NWXCell)

	for i := uint32(0); i < celt.numCells; i++ {
		cellHeader := NewNWXCellHeader()
		if err := cellHeader.Parse(r); err != nil {
			return err
		}
		payloadStart, _ := r.Seek(0, io.SeekCurrent)
		if _, ok := cellCache[int16(cellHeader.index)]; ok {
			continue
		}
		cell := NewNWXCell(payloadStart)
		cellCache[int16(cellHeader.index)] = cell
		if cellHeader.streamW > 0 && cellHeader.streamH > 0 {
			if err := cell.Parse(r, int(cellHeader.streamW), int(cellHeader.streamH), int(cellHeader.size)); err != nil {
				return err
			}
		}
		payloadNext := payloadStart + int64(cellHeader.size)
		if _, err := r.Seek(payloadNext, io.SeekStart); err != nil {
			return err
		}
	}

	for _, frame := range frmt.frames {
		if cell, ok := cellCache[int16(frame.cellIndex)]; ok {
			frame.cell = cell
		}
	}

	for _, action := range w.sequencer.actions {
		for _, node := range action.nodes {
			var logicalFrame *NWXFrame
			if node.index < 0 {
				continue
			}
			if int(node.index) < len(frmt.frames) {
				logicalFrame = frmt.frames[node.index]
			} else if len(frmt.frames) > 0 {
				cleanIdx := int(node.index) % len(frmt.frames)
				logicalFrame = frmt.frames[cleanIdx]
				// TODO Estrazione flag
				//flipX := (node.index & 0x20) != 0
				//fullBright := (node.index & 0x08) != 0
			}
			if logicalFrame != nil {
				node.id = fmt.Sprintf("%s_%d", baseId, node.index)
				node.cell = logicalFrame.cell
			} else {
				fmt.Println("Skipping node with invalid frame index:", baseId, node.index, "max frames", len(frmt.frames))
			}
		}
	}
	return nil
}

// GetActions returns a slice of pointers to NWXAction, representing the actions managed by the NWX sequencer.
func (w *NWX) GetActions() []*NWXAction {
	return w.sequencer.actions
}
