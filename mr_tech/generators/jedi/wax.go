package jedi

import (
	"encoding/binary"
	"fmt"
	"io"
)

// MaxWaxDimension defines the maximum allowable dimension (width or height) for a WAX cell in pixels.
const MaxWaxDimension = 2048

// WaxHeader represents the header structure of a WAX file containing metadata and sequence offsets for actions.
type WaxHeader struct {
	//Version     uint32
	//NumSegments uint32
	//NumFrames   uint32
	//NumCells    uint32
	//ScaleX      uint32
	//ScaleY      uint32
	//ExtraLight  uint32
	//Pad         uint32
	SeqOffsets [32]uint32 // Offset alle Action
}

// NewWaxHeader creates and returns a new instance of WaxHeader with default values.
func NewWaxHeader() *WaxHeader {
	return &WaxHeader{}
}

// Parse reads and parses the WaxHeader data from the given io.ReadSeeker and populates the WaxHeader fields.
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
		SeqOffsets  [32]uint32 // Offset alle Action
	}
	if err := binary.Read(r, binary.LittleEndian, &raw); err != nil {
		return err
	}

	wh.SeqOffsets = raw.SeqOffsets

	return nil
}

// WaxAction represents a structure that holds the offsets for various views (angles) in a 2D animation or rendering context.
type WaxAction struct {
	ViewOffsets [32]uint32 // Offset alle View (Angolazioni)
}

// NewWaxAction creates and initializes a new instance of WaxAction.
func NewWaxAction() *WaxAction {
	return &WaxAction{}
}

// Parse reads and populates the ViewOffsets field from the provided io.ReadSeeker. Returns an error if reading fails.
func (va *WaxAction) Parse(r io.ReadSeeker) error {
	var raw struct {
		Padding     [16]byte
		ViewOffsets [32]uint32 // Offset alle View (Angolazioni)
	}
	if err := binary.Read(r, binary.LittleEndian, &raw); err != nil {
		return err
	}
	va.ViewOffsets = raw.ViewOffsets
	return nil
}

// WaxView represents a structure that stores fixed-size frame offsets for parsing data.
type WaxView struct {
	FrameOffsets [32]uint32
}

// NewWaxView creates and returns a new instance of WaxView with default values.
func NewWaxView() *WaxView {
	return &WaxView{}
}

// Parse reads and extracts frame offset data from the provided io.ReadSeeker and updates the WaxView structure.
func (vw *WaxView) Parse(r io.ReadSeeker) error {
	var raw struct {
		Padding      [16]byte
		FrameOffsets [32]uint32
	}
	if err := binary.Read(r, binary.LittleEndian, &raw); err != nil {
		return err
	}
	vw.FrameOffsets = raw.FrameOffsets
	return nil
}

// WaxFrame represents a frame in the WAX structure, containing positional data, orientation, and a cell reference.
type WaxFrame struct {
	InsertX    int
	InsertY    int
	Flip       bool
	CellOffset uint32
}

// NewWaxFrame creates and returns a pointer to a new, empty WaxFrame instance.
func NewWaxFrame() *WaxFrame {
	return &WaxFrame{}
}

// Parse reads and initializes the WaxFrame fields from the provided io.ReadSeeker, using little-endian byte order.
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
	wf.CellOffset = raw.CellOffset
	wf.InsertX = int(raw.InsertX)
	wf.InsertY = int(raw.InsertY)
	wf.Flip = raw.Flip != 0

	return nil
}

// WaxCellHeader holds metadata for a WAX cell, including dimensions and pixel data in a row-major format.
type WaxCellHeader struct {
	SizeX  int
	SizeY  int
	Pixels []byte
}

// NewWaxCellHeader creates a new instance of WaxCellHeader with default initialization.
func NewWaxCellHeader() *WaxCellHeader {
	return &WaxCellHeader{}
}

// Parse reads and decodes the WaxCellHeader data from an io.ReadSeeker based on the provided WaxFrame information.
func (p *WaxCellHeader) Parse(r io.ReadSeeker, frame *WaxFrame) error {
	offset := int64(frame.CellOffset)
	if _, err := r.Seek(offset, io.SeekStart); err != nil {
		return err
	}

	const headerWaxSize = 24
	// L'header WAX è di soli 24 byte!
	var raw struct {
		SizeX      uint32 // 0-3
		SizeY      uint32 // 4-7
		Compressed uint32 // 8-11
		DataSize   uint32 // 12-15
		ColOffsets uint32 // 16-19 (Spesso 0 in DF)
		Pad1       uint32 // 20-23
	}
	if err := binary.Read(r, binary.LittleEndian, &raw); err != nil {
		return err
	}

	p.SizeX, p.SizeY = int(raw.SizeX), int(raw.SizeY)

	if p.SizeX == 0 || p.SizeY == 0 {
		return nil
	}

	if p.SizeX > MaxWaxDimension || p.SizeY > MaxWaxDimension {
		return fmt.Errorf("cell dimension exceeds maximum: %dx%d", p.SizeX, p.SizeY)
	}

	p.Pixels = make([]byte, p.SizeX*p.SizeY)

	// --- MODALITÀ NON COMPRESSA (RAW) ---
	if raw.Compressed == 0 {
		// Dati RAW partono tipicamente al byte 32 per compatibilità
		rawOffset := offset + 32
		if _, err := r.Seek(rawOffset, io.SeekStart); err != nil {
			return err
		}
		expectedSize := p.SizeX * p.SizeY
		rawData := make([]byte, expectedSize)
		n, err := io.ReadAtLeast(r, rawData, 0)
		if err != nil && err != io.EOF {
			return err
		}
		// Column-Major -> Row-Major
		for x := 0; x < p.SizeX; x++ {
			for y := 0; y < p.SizeY; y++ {
				srcIdx := x*p.SizeY + y
				if srcIdx < n {
					p.Pixels[y*p.SizeX+x] = rawData[srcIdx]
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

	colTable := make([]uint32, p.SizeX)
	if err := binary.Read(r, binary.LittleEndian, colTable); err != nil {
		return fmt.Errorf("failed to read colTable at %d: %w", tableOffset, err)
	}

	for x := 0; x < p.SizeX; x++ {
		if colTable[x] == 0 {
			continue
		}
		// colTable[x] è relativo all'inizio della cella (offset)
		seekPos := offset + int64(colTable[x])
		if _, err := r.Seek(seekPos, io.SeekStart); err != nil {
			continue
		}
		y := 0
		for y < p.SizeY {
			var rawCmd uint8
			if err := binary.Read(r, binary.LittleEndian, &rawCmd); err != nil {
				break
			}
			cmd := int(rawCmd)
			if cmd >= 128 {
				y += cmd - 128
			} else if cmd > 0 {
				count := cmd
				for i := 0; i < count && y < p.SizeY; i++ {
					var pix uint8
					if err := binary.Read(r, binary.LittleEndian, &pix); err == nil {
						p.Pixels[y*p.SizeX+x] = pix // Row-Major nativo
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

// Wax represents a collection of animations, frames, and graphical cells in a structured format for processing.
type Wax struct {
	header  *WaxHeader
	Actions [32]*WaxAction
	Frames  map[uint32]*WaxFrame      // Key: frameOffset
	Cells   map[uint32]*WaxCellHeader // Key: CellOffset
}

// NewWax initializes and returns a new instance of Wax with empty Frames and Cells maps.
func NewWax() *Wax {
	return &Wax{
		Frames: make(map[uint32]*WaxFrame),
		Cells:  make(map[uint32]*WaxCellHeader),
	}
}

// Parse reads and processes the data from the provided io.ReadSeeker, populating the Wax structure with parsed header, actions, views, frames, and cells.
func (w *Wax) Parse(r io.ReadSeeker) error {
	w.header = NewWaxHeader()
	if err := w.header.Parse(r); err != nil {
		return err
	}
	for idx, actOffset := range w.header.SeqOffsets {
		if actOffset == 0 {
			continue
		}
		if _, err := r.Seek(int64(actOffset), io.SeekStart); err != nil {
			return err
		}
		action := NewWaxAction()
		if err := action.Parse(r); err != nil {
			return err
		}
		w.Actions[idx] = action
		for _, viewOffset := range action.ViewOffsets {
			if viewOffset == 0 {
				continue
			}
			if _, err := r.Seek(int64(viewOffset), io.SeekStart); err != nil {
				continue
			}
			view := NewWaxView()
			if err := view.Parse(r); err != nil {
				continue
			}
			for _, frameOffset := range view.FrameOffsets {
				if frameOffset == 0 {
					continue
				}
				if _, exists := w.Frames[frameOffset]; exists {
					continue
				}
				if _, err := r.Seek(int64(frameOffset), io.SeekStart); err != nil {
					continue
				}
				frame := NewWaxFrame()
				if err := frame.Parse(r); err != nil {
					return err
				}
				w.Frames[frameOffset] = frame
				if frame.CellOffset == 0 {
					continue
				}
				if _, exists := w.Cells[frame.CellOffset]; !exists {
					cell := NewWaxCellHeader()
					if err := cell.Parse(r, frame); err != nil {
						fmt.Printf("Skip cell @ %d: %v\n", frame.CellOffset, err)
					} else {
						w.Cells[frame.CellOffset] = cell
					}
				}
			}
		}
	}
	return nil
}
