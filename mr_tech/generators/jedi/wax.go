package jedi

import (
	"encoding/binary"
	"fmt"
	"io"
)

// MaxWaxDimension defines the maximum allowable dimension for wax-related computations or operations.
const MaxWaxDimension = 2048

// --- STRUTTURE DATI ---

// WaxHeader represents the header of a Wax file, containing metadata about sequences, frames, cells, and scaling factors.
type WaxHeader struct {
	Version    uint32
	NumSeqs    uint32
	NumFrames  uint32
	NumCells   uint32
	ScaleX     uint32
	ScaleY     uint32
	ExtraLight uint32
	Pad        uint32
	SeqOffsets [32]uint32 // Offset alle Action
}

// WaxAction represents an action sequence in the WAX data format.
// The Padding field is reserved for future use or alignment.
// The ViewOffsets field contains offsets to different viewing angles.
type WaxAction struct {
	Padding     [16]byte
	ViewOffsets [32]uint32 // Offset alle View (Angolazioni)
}

// WaxView represents a view in a Wax file, containing padding and offsets to associated frames.
type WaxView struct {
	Padding      [16]byte
	FrameOffsets [32]uint32 // Offset ai Frame reali
}

// WaxFrame represents a single frame in the wax format, containing positional, flipping, sizing, and padding metadata.
type WaxFrame struct {
	InsertX    int32  // 0x00
	InsertY    int32  // 0x04
	Flip       int32  // 0x08
	CellOffset uint32 // 0x0C
	UnitWidth  uint32 // 0x10
	UnitHeight uint32 // 0x14
	Pad1       uint32 // 0x18
	Pad2       uint32 // 0x1C
}

// WaxCellHeader represents the header of a cell in a Wax structure, containing metadata and pixel data.
type WaxCellHeader struct {
	SizeX      uint32
	SizeY      uint32
	Compressed uint32
	DataSize   uint32
	ColOffsets uint32
	Padding    [12]byte
	Pixels     []byte
}

// NewWaxCellHeader creates and initializes a new instance of WaxCellHeader with default values.
func NewWaxCellHeader() *WaxCellHeader {
	return &WaxCellHeader{}
}

// Parse reads and processes data from the provided io.ReadSeeker starting at the specified offset to populate the struct.
func (p *WaxCellHeader) Parse(r io.ReadSeeker, offset int64) error {
	if _, err := r.Seek(offset, io.SeekStart); err != nil {
		return err
	}

	// 1. Leggiamo l'header fisso di 32 byte
	var raw struct {
		SizeX, SizeY, Compressed, DataSize uint32 // 0-15
		Reserved1, Reserved2               uint32 // 16-23
		ColOffsets                         uint32 // 24-27
		Reserved3                          uint32 // 28-31
	}
	if err := binary.Read(r, binary.LittleEndian, &raw); err != nil {
		return err
	}

	p.SizeX, p.SizeY = raw.SizeX, raw.SizeY
	p.Compressed = raw.Compressed
	p.DataSize = raw.DataSize

	if p.SizeX == 0 || p.SizeY == 0 {
		return nil
	}

	if p.SizeX > MaxWaxDimension || p.SizeY > MaxWaxDimension {
		return fmt.Errorf("cell dimension exceeds maximum allowed size: %dx%d", p.SizeX, p.SizeY)
	}

	// Determiniamo dove inizia effettivamente la tabella delle colonne
	// Se ColOffsets è 0, la tabella segue l'header (byte 32)
	tableStartRelative := int64(raw.ColOffsets)
	if tableStartRelative == 0 {
		tableStartRelative = 32
	}

	p.Pixels = make([]byte, p.SizeX*p.SizeY)

	if p.Compressed == 0 {
		if _, err := r.Seek(offset+tableStartRelative, io.SeekStart); err != nil {
			return err
		}
		//rawData := make([]byte, p.SizeX*p.SizeY)
		/*
			n, _ := io.ReadAtLeast(r, rawData, 0)
			for x := uint32(0); x < p.SizeX; x++ {
				for y := uint32(0); y < p.SizeY; y++ {
					idx := x*p.SizeY + y
					if idx < uint32(n) {
						p.Pixels[y*p.SizeX+x] = rawData[idx]
					}
				}
			}

		*/
		rawData := make([]byte, p.SizeX*p.SizeY)
		total := 0
		for total < len(rawData) {
			n, err := r.Read(rawData[total:])
			total += n
			if err != nil {
				if err == io.EOF {
					break // EOF parziale
				}
				return err
			}
		}
		if total < len(rawData) {
			for i := total; i < len(rawData); i++ {
				rawData[i] = 0
			}
		}
		return nil
	}

	// --- MODALITÀ COMPRESSA (RLE) ---
	// Leggiamo la tabella degli offset (uno per colonna)
	if _, err := r.Seek(offset+tableStartRelative, io.SeekStart); err != nil {
		return err
	}
	colTable := make([]uint32, p.SizeX)
	if err := binary.Read(r, binary.LittleEndian, colTable); err != nil {
		return fmt.Errorf("failed to read colTable at %d: %w", offset+tableStartRelative, err)
	}

	for x := uint32(0); x < p.SizeX; x++ {
		if colTable[x] == 0 {
			continue
		}
		if _, err := r.Seek(offset+int64(colTable[x]), io.SeekStart); err != nil {
			continue
		}
		y := uint32(0)
		for y < p.SizeY {
			var cmd uint8
			if err := binary.Read(r, binary.LittleEndian, &cmd); err != nil {
				break // EOF su questa colonna, passa alla prossima
			}
			if cmd >= 128 {
				// Skip (Trasparenza)
				y += uint32(cmd - 128)
			} else if cmd > 0 {
				// Pixel opachi
				count := uint32(cmd)
				for i := uint32(0); i < count && y < p.SizeY; i++ {
					var pix uint8
					if binary.Read(r, binary.LittleEndian, &pix) == nil {
						p.Pixels[y*p.SizeX+x] = pix
					}
					y++
				}
			} else {
				break // cmd 0 = fine colonna
			}
		}
	}
	return nil
}

// Wax represents a structured data container holding header information, actions, frames, and cell headers.
type Wax struct {
	Header  WaxHeader
	Actions [32]*WaxAction
	Frames  map[uint32]*WaxFrame      // Key: frameOffset
	Cells   map[uint32]*WaxCellHeader // Key: CellOffset
}

// NewWax creates and returns a new initialized instance of Wax with empty Frames and Cells maps.
func NewWax() *Wax {
	return &Wax{
		Frames: make(map[uint32]*WaxFrame),
		Cells:  make(map[uint32]*WaxCellHeader),
	}
}

// Parse reads and parses the Wax file structure from the given io.ReadSeeker, populating the Wax object with its components.
// Returns an error if the parsing operation fails at any stage.
func (w *Wax) Parse(r io.ReadSeeker) error {
	if err := binary.Read(r, binary.LittleEndian, &w.Header); err != nil {
		return err
	}
	// Iteriamo sulle Action (Sequenze)
	for idx, actOffset := range w.Header.SeqOffsets {
		if actOffset == 0 {
			continue
		}
		if _, err := r.Seek(int64(actOffset), io.SeekStart); err != nil {
			return err
		}
		action := &WaxAction{}
		// Usiamo un controllo meno rigido sulla lettura delle tabelle
		if err := binary.Read(r, binary.LittleEndian, action); err != nil {
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
			view := &WaxView{}
			if err := binary.Read(r, binary.LittleEndian, view); err != nil {
				continue
			}
			// Iteriamo sui Frame
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
				frame := &WaxFrame{}
				if err := binary.Read(r, binary.LittleEndian, frame); err != nil {
					continue
				}
				w.Frames[frameOffset] = frame
				if frame.CellOffset == 0 {
					continue
				}
				if _, exists := w.Cells[frame.CellOffset]; !exists {
					cell := NewWaxCellHeader()
					if err := cell.Parse(r, int64(frame.CellOffset)); err != nil {
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
