package jedi

import (
	"encoding/binary"
	"fmt"
	"io"
)

// MaxWaxDimension defines the maximum allowed dimension for a WAX cell to ensure data validity and prevent corruption.
const MaxWaxDimension = 2048

// --- STRUTTURE DATI ---

// WaxHeader represents the metadata for a WAX file, including information about sequences, frames, and scaling factors.
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

// WaxAction represents an action within the WAX file format, containing padding and offsets to associated views.
type WaxAction struct {
	Padding     [16]byte
	ViewOffsets [32]uint32 // Offset alle View (Angolazioni)
}

// WaxView represents a view in a WAX file, containing padding and offsets to associated frames.
type WaxView struct {
	Padding      [16]byte
	FrameOffsets [32]uint32 // Offset ai Frame reali
}

// WaxFrame represents a single frame in a WAX file, containing metadata and offsets to pixel data.
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

// WaxCellHeader represents metadata and pixel data for a single cell in a WAX file, including compression and layout details.
type WaxCellHeader struct {
	SizeX      uint32
	SizeY      uint32
	Compressed uint32
	DataSize   uint32
	ColOffsets uint32
	Padding    [12]byte
	Pixels     []byte
}

// Parse reads and initializes the WaxCellHeader structure from the provided io.ReadSeeker at the given offset.
func (p *WaxCellHeader) Parse(r io.ReadSeeker, offset int64) error {
	if _, err := r.Seek(offset, io.SeekStart); err != nil {
		return err
	}
	var raw struct {
		SizeX,
		SizeY,
		Compressed,
		DataSize,
		ColOffsets uint32
		Padding [12]byte
	}
	if err := binary.Read(r, binary.LittleEndian, &raw); err != nil {
		return err
	}

	p.SizeX, p.SizeY = raw.SizeX, raw.SizeY
	p.Compressed, p.ColOffsets = raw.Compressed, raw.ColOffsets

	if p.SizeX == 0 || p.SizeY == 0 {
		return nil
	}

	if p.SizeX > MaxWaxDimension || p.SizeY > MaxWaxDimension {
		return fmt.Errorf("invalid dimensions: %dx%d", p.SizeX, p.SizeY)
	}

	p.Pixels = make([]byte, p.SizeX*p.SizeY)

	//se non ci sono offset, abbiamo finito (cella trasparente)
	if p.ColOffsets == 0 {
		return nil
	}

	dataOffset := offset + int64(p.ColOffsets)
	if _, err := r.Seek(dataOffset, io.SeekStart); err != nil {
		return err
	}

	if p.Compressed == 0 {
		// Se non compresso, leggiamo SizeX * SizeY
		rawData := make([]byte, p.SizeX*p.SizeY)
		if _, err := io.ReadFull(r, rawData); err != nil {
			return fmt.Errorf("EOF su raw pixels: %w", err)
		}
		// Trasposizione Column-Major -> Row-Major
		for x := uint32(0); x < p.SizeX; x++ {
			for y := uint32(0); y < p.SizeY; y++ {
				p.Pixels[y*p.SizeX+x] = rawData[x*p.SizeY+y]
			}
		}
		return nil
	}

	// --- RLE Variate (LSB Flag) ---
	colOffsets := make([]uint32, p.SizeX)
	if err := binary.Read(r, binary.LittleEndian, colOffsets); err != nil {
		return err
	}

	for x := uint32(0); x < p.SizeX; x++ {
		if _, err := r.Seek(offset+int64(colOffsets[x]), io.SeekStart); err != nil {
			continue
		}
		y := uint32(0)
		for y < p.SizeY {
			var cmd uint8
			if err := binary.Read(r, binary.LittleEndian, &cmd); err != nil {
				break // Fine colonna
			}
			if cmd == 0 {
				break
			}

			count := uint32(cmd >> 1)
			if (cmd & 0x01) == 0 { // SKIP
				y += count
			} else {
				for i := uint32(0); i < count; i++ {
					var pix uint8
					if binary.Read(r, binary.LittleEndian, &pix) == nil && y < p.SizeY {
						p.Pixels[y*p.SizeX+x] = pix
					}
					y++
				}
			}
		}
	}
	return nil
}

// Wax represents the main structure of a WAX file, which organizes graphical data for animations and frames.
type Wax struct {
	Header  WaxHeader
	Actions [32]*WaxAction
	Frames  map[uint32]*WaxFrame      // Key: frameOffset
	Cells   map[uint32]*WaxCellHeader // Key: CellOffset
}

// NewWax initializes and returns a pointer to a new Wax instance with empty maps for Frames and Cells.
func NewWax() *Wax {
	return &Wax{
		Frames: make(map[uint32]*WaxFrame),
		Cells:  make(map[uint32]*WaxCellHeader),
	}
}

// Parse reads and parses the WAX structure, including actions, views, frames, and cells, from the provided io.ReadSeeker.
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
			continue
		}

		action := &WaxAction{}
		// Usiamo un controllo meno rigido sulla lettura delle tabelle
		if err := binary.Read(r, binary.LittleEndian, action); err != nil {
			continue
		}
		w.Actions[idx] = action

		// Iteriamo sulle View (Angolazioni)
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

				// Parsing della Cella
				if frame.CellOffset != 0 {
					if _, exists := w.Cells[frame.CellOffset]; !exists {
						cell := &WaxCellHeader{}
						// Se la cella fallisce, logghiamo ma non interrompiamo il file
						if err := cell.Parse(r, int64(frame.CellOffset)); err != nil {
							fmt.Printf("Skip cell @ %d: %v\n", frame.CellOffset, err)
						} else {
							w.Cells[frame.CellOffset] = cell
						}
					}
				}
			}
		}
	}
	return nil
}
