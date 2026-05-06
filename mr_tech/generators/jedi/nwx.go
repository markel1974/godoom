package jedi

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

// NWXChunk represents a data chunk with an offset and size within a binary structure.
type NWXChunk struct {
	Offset int64
	Size   uint32
}

// mapNWXChunks reads a WAXF file from the provided io.ReadSeeker and maps its chunks by name into an NWXChunk map.
func mapNWXChunks(r io.ReadSeeker) (map[string]NWXChunk, error) {
	chunks := make(map[string]NWXChunk)

	var sig [4]byte
	if err := binary.Read(r, binary.LittleEndian, &sig); err != nil {
		return nil, err
	}

	if string(sig[:]) != "WAXF" {
		return nil, fmt.Errorf("invalid signature, expected WAXF got %s", string(sig[:]))
	}

	var totalSize uint32
	if err := binary.Read(r, binary.LittleEndian, &totalSize); err != nil {
		return nil, err
	}

	for {
		var tag [4]byte
		err := binary.Read(r, binary.LittleEndian, &tag)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		var size uint32
		if err := binary.Read(r, binary.LittleEndian, &size); err != nil {
			return nil, err
		}

		pos, _ := r.Seek(0, io.SeekCurrent)
		chunkName := strings.TrimRight(string(tag[:]), "\x00 ")
		chunks[chunkName] = NWXChunk{
			Offset: pos,
			Size:   size,
		}

		// Salta al prossimo chunk
		if _, err := r.Seek(int64(size), io.SeekCurrent); err != nil {
			break
		}
	}

	return chunks, nil
}

// NWXCell represents a graphical cell containing pixel data, dimensions, and an associated frame configuration.
type NWXCell struct {
	id     string
	sizeX  int
	sizeY  int
	pixels []byte
	frame  *NWXFrame
}

// GetId returns the unique identifier of the NWXCell instance.
func (p *NWXCell) GetId() string { return p.id }

// GetSize returns the width and height of the cell as two integers.
func (p *NWXCell) GetSize() (int, int) { return p.sizeX, p.sizeY }

// GetPixels returns the pixel data of the NWXCell as a byte slice.
func (p *NWXCell) GetPixels() []byte { return p.pixels }

// GetFrame returns the NWXFrame object associated with the NWXCell instance.
func (p *NWXCell) GetFrame() *NWXFrame { return p.frame }

// NewNWXCell creates a new instance of NWXCell with the given id and associated NWXFrame.
func NewNWXCell(id string, frame *NWXFrame) *NWXCell {
	return &NWXCell{id: id, frame: frame}
}

// Clone creates and returns a new NWXCell object with the same properties as the current instance.
func (p *NWXCell) Clone() *NWXCell {
	return &NWXCell{
		id:     p.id,
		sizeX:  p.sizeX,
		sizeY:  p.sizeY,
		pixels: p.pixels,
		frame:  p.frame,
	}
}

// Parse reads, processes, and populates NWXCell data from a given io.ReadSeeker based on the specified base offset.
func (p *NWXCell) Parse(r io.ReadSeeker, celtBaseOffset int64) error {
	// FIX: L'offset della cella è relativo all'inizio del chunk CELT
	offset := celtBaseOffset + int64(p.frame.cellOffset)
	if _, err := r.Seek(offset, io.SeekStart); err != nil {
		return err
	}

	const headerNWXSize = 24
	const maxNWXDimension = 2048

	var raw struct {
		SizeX      uint32
		SizeY      uint32
		Compressed uint32
		DataSize   uint32
		ColOffsets uint32
		Pad1       uint32
	}
	if err := binary.Read(r, binary.LittleEndian, &raw); err != nil {
		return err
	}

	p.sizeX, p.sizeY = int(raw.SizeX), int(raw.SizeY)
	if p.sizeX == 0 || p.sizeY == 0 {
		return nil
	}

	if p.sizeX > maxNWXDimension || p.sizeY > maxNWXDimension {
		return fmt.Errorf("cell dimension exceeds maximum: %dx%d", p.sizeX, p.sizeY)
	}

	size := p.sizeX * p.sizeY
	p.pixels = make([]byte, size)

	if raw.Compressed == 0 {
		dataOffset := offset + headerNWXSize
		if raw.ColOffsets != 0 {
			dataOffset = offset + int64(raw.ColOffsets)
		}
		if _, err := r.Seek(dataOffset, io.SeekStart); err != nil {
			return err
		}
		rawData := make([]byte, size)
		n := 0
		for n < size {
			nn, err := r.Read(rawData[n:])
			n += nn
			if err != nil {
				break
			}
		}
		if n == 0 {
			return nil
		}
		for x := 0; x < p.sizeX; x++ {
			for y := 0; y < p.sizeY; y++ {
				srcIdx := x*p.sizeY + y
				if srcIdx < n {
					p.pixels[y*p.sizeX+x] = rawData[srcIdx]
				}
			}
		}
		return nil
	}

	// COMPRESSED MODE
	tableOffset := offset + headerNWXSize
	if _, err := r.Seek(tableOffset, io.SeekStart); err != nil {
		return err
	}

	colTable := make([]uint32, p.sizeX)
	if err := binary.Read(r, binary.LittleEndian, colTable); err != nil {
		return err
	}

	for x := 0; x < p.sizeX; x++ {
		if colTable[x] == 0 {
			continue
		}
		seekPos := offset + int64(colTable[x])
		if _, err := r.Seek(seekPos, io.SeekStart); err != nil {
			continue
		}
		y := 0
		for y < p.sizeY {
			var rawCmd uint8
			if err := binary.Read(r, binary.LittleEndian, &rawCmd); err != nil {
				break
			}
			cmd := int(rawCmd)
			if cmd >= 128 {
				y += cmd - 128
			} else if cmd > 0 {
				for i := 0; i < cmd && y < p.sizeY; i++ {
					var pix uint8
					if err := binary.Read(r, binary.LittleEndian, &pix); err == nil {
						p.pixels[y*p.sizeX+x] = pix
					}
					y++
				}
			} else {
				break
			}
		}
	}
	return nil
}

// NWXView represents a data structure containing a unique identifier, a cache of NWXCell objects, and a list of NWXCell instances.
type NWXView struct {
	id        string
	cellCache map[uint32]*NWXCell
	cells     []*NWXCell
}

// NewNWXView creates and returns a new NWXView instance with the provided id and cellCache.
func NewNWXView(id string, cellCache map[uint32]*NWXCell) *NWXView {
	return &NWXView{id: id, cellCache: cellCache}
}

// GetCells returns a slice of NWXCell pointers stored in the NWXView. It provides access to the parsed cell data.
func (vw *NWXView) GetCells() []*NWXCell { return vw.cells }

// Parse parses an NWXView object using data from the provided ReadSeeker and chunk map, populating the view with its cells.
func (vw *NWXView) Parse(r io.ReadSeeker, chunks map[string]NWXChunk) error {
	var raw struct {
		Padding      [16]byte
		FrameOffsets [32]uint32
	}
	if err := binary.Read(r, binary.LittleEndian, &raw); err != nil {
		return err
	}

	frmtChunk, hasFrmt := chunks["FRMT"]
	celtChunk, hasCelt := chunks["CELT"]
	if !hasFrmt || !hasCelt {
		return fmt.Errorf("missing FRMT or CELT chunk")
	}

	for _, frameOffset := range raw.FrameOffsets {
		if frameOffset == 0 {
			continue
		}
		// FIX: Offset del frame relativo a FRMT
		if _, err := r.Seek(frmtChunk.Offset+int64(frameOffset), io.SeekStart); err != nil {
			continue
		}
		frame := NewNWXFrame()
		if err := frame.Parse(r); err != nil {
			return err
		}
		if frame.cellOffset == 0 {
			continue
		}
		if cachedCell, exists := vw.cellCache[frame.cellOffset]; exists {
			cloned := cachedCell.Clone()
			cloned.frame = frame
			vw.cells = append(vw.cells, cloned)
			continue
		}
		cellId := fmt.Sprintf("%s_frame_%d", vw.id, frame.cellOffset)
		cell := NewNWXCell(cellId, frame)

		// Passiamo l'offset del chunk CELT per il calcolo della posizione in memoria
		if err := cell.Parse(r, celtChunk.Offset); err != nil {
			fmt.Printf("Skip cell @ %d: %v\n", frame.cellOffset, err)
			continue
		}
		vw.cellCache[frame.cellOffset] = cell
		vw.cells = append(vw.cells, cell)
	}
	return nil
}

// NWXFrame represents a single frame containing metadata for graphical cell positioning and orientation.
type NWXFrame struct {
	insertX    int
	insertY    int
	flip       bool
	cellOffset uint32
}

// NewNWXFrame creates a new instance of NWXFrame with default values.
func NewNWXFrame() *NWXFrame { return &NWXFrame{} }

// Parse reads and decodes binary data from the provided io.ReadSeeker into the NWXFrame instance.
func (wf *NWXFrame) Parse(r io.ReadSeeker) error {
	var raw struct {
		InsertX    int32
		InsertY    int32
		Flip       int32
		CellOffset uint32
		UnitWidth  uint32
		UnitHeight uint32
		Pad1       uint32
		Pad2       uint32
	}
	if err := binary.Read(r, binary.LittleEndian, &raw); err != nil {
		return err
	}
	wf.cellOffset = raw.CellOffset
	wf.insertX = int(raw.InsertX)
	wf.insertY = int(raw.InsertY)
	wf.flip = raw.Flip != 0
	return nil
}

// NWXActions represents a collection of views associated with an action, leveraging cached cells for rendering.
type NWXActions struct {
	id        string
	cellCache map[uint32]*NWXCell
	views     [32]*NWXView
}

// NewNWXActions creates a new NWXActions instance with a given ID and cell cache.
func NewNWXActions(id string, cellCache map[uint32]*NWXCell) *NWXActions {
	return &NWXActions{id: id, cellCache: cellCache}
}

// GetViews returns an array of pointers to NWXView objects held within the NWXActions instance.
func (wa *NWXActions) GetViews() [32]*NWXView { return wa.views }

// Parse reads and processes NWX action data from the provided io.ReadSeeker and populates associated views using chunk mapping.
func (wa *NWXActions) Parse(r io.ReadSeeker, chunks map[string]NWXChunk) error {
	var raw struct {
		Padding     [16]byte
		ViewOffsets [32]uint32
	}
	if err := binary.Read(r, binary.LittleEndian, &raw); err != nil {
		return err
	}

	chotChunk := chunks["CHOT"]

	for idx, viewOffset := range raw.ViewOffsets {
		if viewOffset == 0 {
			continue
		}
		// FIX: Offset della View relativo a CHOT
		if _, err := r.Seek(chotChunk.Offset+int64(viewOffset), io.SeekStart); err != nil {
			continue
		}
		viewId := fmt.Sprintf("%s_view_%d", wa.id, idx)
		view := NewNWXView(viewId, wa.cellCache)
		if err := view.Parse(r, chunks); err != nil {
			continue
		}
		wa.views[idx] = view
	}
	return nil
}

// NWXHeader represents the header structure of an NWX file containing sequence offsets.
type NWXHeader struct {
	seqOffsets [32]uint32
}

// NewNWXHeader creates and returns a new instance of NWXHeader with default values.
func NewNWXHeader() *NWXHeader { return &NWXHeader{} }

// Parse reads and parses NWXHeader data from the provided io.ReadSeeker and initializes the seqOffsets field.
func (wh *NWXHeader) Parse(r io.ReadSeeker) error {
	var raw struct {
		Version     uint32
		NumSegments uint32
		NumFrames   uint32
		NumCells    uint32
		ScaleX      uint32
		ScaleY      uint32
		ExtraLight  uint32
		Pad         uint32
		SeqOffsets  [32]uint32
	}
	if err := binary.Read(r, binary.LittleEndian, &raw); err != nil {
		return err
	}
	wh.seqOffsets = raw.SeqOffsets
	return nil
}

// NWX represents the root structure for handling NWX format data, including its header, actions, and associated chunks.
type NWX struct {
	header  *NWXHeader
	actions []*NWXActions
	chunks  map[string]NWXChunk
}

// NewNWX creates and returns a new instance of the NWX structure.
func NewNWX() *NWX { return &NWX{} }
func (w *NWX) Parse(baseId string, r io.ReadSeeker) error {
	chunks, err := mapNWXChunks(r)
	if err != nil {
		return fmt.Errorf("failed to map chunks: %w", err)
	}
	w.chunks = chunks

	// DEBUG: Stampa i chunk per capire l'architettura del file
	fmt.Printf("--- Chunks in %s ---\n", baseId)
	for name, c := range chunks {
		fmt.Printf(" - [%s] Offset: %d, Size: %d\n", name, c.Offset, c.Size)
	}

	// Risoluzione dinamica del blocco contenente l'Header / Sequenze
	var seqChunk NWXChunk
	var ok bool

	// Tentiamo i FourCC più comuni usati da LucasArts
	validSeqChunks := []string{"CHOT", "SEQT", "ACTN", "HEAD", "ANIM"}
	for _, tag := range validSeqChunks {
		if seqChunk, ok = chunks[tag]; ok {
			break
		}
	}

	if !ok {
		return fmt.Errorf("missing sequence chunk in WAXF file. Found chunks: %v", chunks)
	}

	// Usiamo il chunk trovato (seqChunk) invece dell'hardcoded chotChunk
	if _, err := r.Seek(seqChunk.Offset, io.SeekStart); err != nil {
		return err
	}

	w.header = NewNWXHeader()
	if err := w.header.Parse(r); err != nil {
		return err
	}

	cellCache := make(map[uint32]*NWXCell)
	for _, actOffset := range w.header.seqOffsets {
		if actOffset == 0 {
			continue
		}
		// Offset relativo al blocco di sequenza dinamico
		if _, err := r.Seek(seqChunk.Offset+int64(actOffset), io.SeekStart); err != nil {
			return err
		}
		actorId := fmt.Sprintf("%s_act_%d", baseId, actOffset)
		action := NewNWXActions(actorId, cellCache)
		if err := action.Parse(r, chunks); err != nil {
			return err
		}
		w.actions = append(w.actions, action)
	}
	return nil
}

// GetActions retrieves a list of NWXActions associated with the NWX instance.
func (w *NWX) GetActions() []*NWXActions { return w.actions }
