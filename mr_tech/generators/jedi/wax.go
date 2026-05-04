package jedi

import (
	"encoding/binary"
	"fmt"
	"io"
)

// MaxWaxDimension defines the maximum allowed dimension for the width or height of a WaxCell (in pixels).
const MaxWaxDimension = 2048

// WaxCell represents a graphical cell with associated metadata, pixel data, and a reference to its parent WaxFrame.
type WaxCell struct {
	id     string
	sizeX  int
	sizeY  int
	pixels []byte
	frame  *WaxFrame
}

// GetId returns the identifier of the WaxCell as a string.
func (p *WaxCell) GetId() string {
	return p.id
}

// GetSize returns the dimensions of the WaxCell as two integers: sizeX and sizeY.
func (p *WaxCell) GetSize() (int, int) {
	return p.sizeX, p.sizeY
}

// GetPixels returns the pixel data associated with the WaxCell as a slice of bytes.
func (p *WaxCell) GetPixels() []byte {
	return p.pixels
}

// GetFrame retrieves the associated WaxFrame of the current WaxCell instance.
func (p *WaxCell) GetFrame() *WaxFrame {
	return p.frame
}

// NewWaxCell creates a new WaxCell instance with the specified id and associated WaxFrame.
func NewWaxCell(id string, frame *WaxFrame) *WaxCell {
	return &WaxCell{
		id:    id,
		frame: frame,
	}
}

// Clone creates a deep copy of the current WaxCell, including its properties and associated frame data.
func (p *WaxCell) Clone() *WaxCell {
	return &WaxCell{
		id:     p.id,
		sizeX:  p.sizeX,
		sizeY:  p.sizeY,
		pixels: p.pixels,
		frame:  p.frame,
	}
}

// Parse reads and processes the WAX cell data from the given io.ReadSeeker, handling both compressed and uncompressed formats.
func (p *WaxCell) Parse(r io.ReadSeeker) error {
	offset := int64(p.frame.cellOffset)
	if _, err := r.Seek(offset, io.SeekStart); err != nil {
		return err
	}
	const headerWaxSize = 24 // L'header WAX è di soli 24 byte!
	var raw struct {
		SizeX      uint32 // 0-3
		SizeY      uint32 // 4-7
		Compressed uint32 // 8-11
		DataSize   uint32 // 12-15
		ColOffsets uint32 // 16-19
		Pad1       uint32 // 20-23
	}
	if err := binary.Read(r, binary.LittleEndian, &raw); err != nil {
		return err
	}

	p.sizeX, p.sizeY = int(raw.SizeX), int(raw.SizeY)
	if p.sizeX == 0 || p.sizeY == 0 {
		return nil
	}

	if p.sizeX > MaxWaxDimension || p.sizeY > MaxWaxDimension {
		return fmt.Errorf("cell dimension exceeds maximum: %dx%d", p.sizeX, p.sizeY)
	}

	size := p.sizeX * p.sizeY
	p.pixels = make([]byte, size)

	if raw.Compressed == 0 {
		dataOffset := offset + headerWaxSize
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
		// Column-Major -> Row-Major
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

	// --- MODALITÀ COMPRESSA (RLE) ---
	tableOffset := offset + headerWaxSize
	if _, err := r.Seek(tableOffset, io.SeekStart); err != nil {
		return err
	}

	colTable := make([]uint32, p.sizeX)
	if err := binary.Read(r, binary.LittleEndian, colTable); err != nil {
		return fmt.Errorf("failed to read colTable at %d: %w", tableOffset, err)
	}

	for x := 0; x < p.sizeX; x++ {
		if colTable[x] == 0 {
			continue
		}
		// colTable[x] è relativo all'inizio della cella (offset)
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
				count := cmd
				for i := 0; i < count && y < p.sizeY; i++ {
					var pix uint8
					if err := binary.Read(r, binary.LittleEndian, &pix); err == nil {
						p.pixels[y*p.sizeX+x] = pix // Row-Major nativo
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

// WaxView represents a collection of WaxCells and manages their data and lifecycle.
type WaxView struct {
	id        string
	cellCache map[uint32]*WaxCell
	cells     []*WaxCell
}

// NewWaxView creates a new WaxView with the specified ID and a shared cell cache map.
func NewWaxView(id string, cellCache map[uint32]*WaxCell) *WaxView {
	return &WaxView{
		id:        id,
		cellCache: cellCache,
	}
}

// GetCells returns a slice of pointers to WaxCell instances associated with the WaxView.
func (vw *WaxView) GetCells() []*WaxCell {
	return vw.cells
}

// Parse reads the WaxView data structure from the provided io.ReadSeeker, parsing frames and their corresponding wax cells.
func (vw *WaxView) Parse(r io.ReadSeeker) error {
	var raw struct {
		Padding      [16]byte
		FrameOffsets [32]uint32
	}
	if err := binary.Read(r, binary.LittleEndian, &raw); err != nil {
		return err
	}

	for _, frameOffset := range raw.FrameOffsets {
		if frameOffset == 0 {
			continue
		}
		if _, err := r.Seek(int64(frameOffset), io.SeekStart); err != nil {
			continue
		}
		frame := NewWaxFrame()
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
		cell := NewWaxCell(cellId, frame)
		if err := cell.Parse(r); err != nil {
			fmt.Printf("Skip cell @ %d: %v\n", frame.cellOffset, err)
			continue
		}
		vw.cellCache[frame.cellOffset] = cell
		vw.cells = append(vw.cells, cell)
	}
	return nil
}

// WaxFrame represents frame metadata, including positional offsets, flipping state, and cell data offset in memory.
type WaxFrame struct {
	insertX    int
	insertY    int
	flip       bool
	cellOffset uint32
}

// NewWaxFrame creates and returns a pointer to a new, empty instance of WaxFrame.
func NewWaxFrame() *WaxFrame {
	return &WaxFrame{}
}

// Parse reads and populates the WaxFrame fields from the provided io.ReadSeeker in little-endian binary format.
func (wf *WaxFrame) Parse(r io.ReadSeeker) error {
	var raw struct {
		InsertX    int32  // 0x00
		InsertY    int32  // 0x04
		Flip       int32  // 0x08
		CellOffset uint32 // 0x0C
		UnitWidth  uint32 // 0x10
		UnitHeight uint32 // 0x14
		Pad1       uint32 // 0x18
		Pad2       uint32 // 0x1C
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

// WaxActions represents an entity that manages WaxView objects and a shared cache of WaxCells.
type WaxActions struct {
	id        string
	cellCache map[uint32]*WaxCell
	views     [32]*WaxView
}

// NewWaxActions creates a new WaxActions instance with a unique identifier and a shared cell cache.
func NewWaxActions(id string, cellCache map[uint32]*WaxCell) *WaxActions {
	return &WaxActions{
		id:        id,
		cellCache: cellCache,
	}
}

// GetViews returns an array of pointers to WaxView objects associated with the WaxActor.
func (wa *WaxActions) GetViews() [32]*WaxView {
	return wa.views
}

// Parse reads and processes data from the provided io.ReadSeeker to populate the WaxActor's views collection.
func (wa *WaxActions) Parse(r io.ReadSeeker) error {
	var raw struct {
		Padding     [16]byte
		ViewOffsets [32]uint32
	}
	if err := binary.Read(r, binary.LittleEndian, &raw); err != nil {
		return err
	}

	for idx, viewOffset := range raw.ViewOffsets {
		if viewOffset == 0 {
			continue
		}
		if _, err := r.Seek(int64(viewOffset), io.SeekStart); err != nil {
			continue
		}
		viewId := fmt.Sprintf("%s_view_%d", wa.id, idx)
		view := NewWaxView(viewId, wa.cellCache)
		if err := view.Parse(r); err != nil {
			continue
		}
		wa.views[idx] = view
	}
	return nil
}

// WaxHeader represents a structure for holding sequence offset data for actions within a WAX binary.
type WaxHeader struct {
	seqOffsets [32]uint32
}

// NewWaxHeader initializes and returns a pointer to a new WaxHeader instance.
func NewWaxHeader() *WaxHeader {
	return &WaxHeader{}
}

// Parse reads and populates the fields of the WaxHeader from the given io.ReadSeeker data source.
func (wh *WaxHeader) Parse(r io.ReadSeeker) error {
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

// Wax represents a container for managing header and actions in a game asset or animation format.
type Wax struct {
	header  *WaxHeader
	actions []*WaxActions
}

// NewWax creates and returns a new instance of the Wax structure.
func NewWax() *Wax {
	return &Wax{}
}

// Parse initializes the Wax object by reading and parsing its header and associated actions from the provided ReadSeeker.
func (w *Wax) Parse(baseId string, r io.ReadSeeker) error {
	w.header = NewWaxHeader()
	if err := w.header.Parse(r); err != nil {
		return err
	}
	cellCache := make(map[uint32]*WaxCell)
	for _, actOffset := range w.header.seqOffsets {
		if actOffset == 0 {
			continue
		}
		if _, err := r.Seek(int64(actOffset), io.SeekStart); err != nil {
			return err
		}
		actorId := fmt.Sprintf("%s_act_%d", baseId, actOffset)
		action := NewWaxActions(actorId, cellCache)
		if err := action.Parse(r); err != nil {
			return err
		}
		w.actions = append(w.actions, action)
	}
	return nil
}

// GetActions returns the slice of WaxActor pointers representing the actions associated with the Wax object.
func (w *Wax) GetActions() []*WaxActions {
	return w.actions
}
