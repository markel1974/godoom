package jedi

import (
	"encoding/binary"
	"fmt"
	"io"
	"unsafe"
)

// NWXCell represents a graphical cell containing pixel data and metadata.
// It includes the cell's ID, dimensions, pixel array, and an associated frame reference.
type NWXCell struct {
	id     string
	sizeX  int
	sizeY  int
	pixels []byte
}

// GetId returns the unique identifier of the NWXCell.
func (p *NWXCell) GetId() string { return p.id }

// GetSize returns the width and height of the NWXCell in pixels.
func (p *NWXCell) GetSize() (int, int) { return p.sizeX, p.sizeY }

// GetPixels returns the raw pixel data of the NWXCell as a byte slice.
func (p *NWXCell) GetPixels() []byte { return p.pixels }

// NewNWXCell creates a new NWXCell instance with the given ID and associated NWXFrame.
func NewNWXCell(id string) *NWXCell {
	return &NWXCell{id: id}
}

// Parse reads and decodes pixel data from an io.ReadSeeker into the NWXCell, given the specified stream dimensions.
// Parse decodifica i dati della cella usando la metodologia passata dall'esterno (RAW o RLE).
func (p *NWXCell) Parse(r io.ReadSeeker, streamW, streamH int, isCompressed bool) error {
	p.sizeX, p.sizeY = streamW, streamH
	p.pixels = make([]byte, streamW*streamH)

	colTableBase, _ := r.Seek(0, io.SeekCurrent)

	colTable := make([]uint32, streamW)
	if err := binary.Read(r, binary.LittleEndian, &colTable); err != nil {
		return err
	}

	for x := 0; x < streamW; x++ {
		target := colTable[x] & 0x00FFFFFF
		if target == 0 {
			continue
		}

		if _, err := r.Seek(colTableBase+int64(target), io.SeekStart); err != nil {
			return err
		}

		if !isCompressed {
			// ==========================================
			// BRANCH RAW: Dati lineari (es. Barile)
			// ==========================================
			for y := 0; y < streamH; y++ {
				var pix uint8
				if err := binary.Read(r, binary.LittleEndian, &pix); err != nil {
					break
				}

				if pix == 0 {
					continue // Trasparenza implicita (indice 0)
				}

				drawY := (streamH - 1) - y
				targetIdx := (drawY * streamW) + x
				if targetIdx >= 0 && targetIdx < len(p.pixels) {
					p.pixels[targetIdx] = pix
				}
			}
		} else {
			// ==========================================
			// BRANCH RLE: Dati compressi con terminatori
			// ==========================================
			y := 0
			for y < streamH {
				var cmd uint8
				if err := binary.Read(r, binary.LittleEndian, &cmd); err != nil {
					break
				}

				// Terminatori di colonna del motore LucasArts
				if cmd == 0 || cmd == 128 {
					break
				}

				if cmd > 128 {
					y += int(cmd - 128) // Salto (Trasparenza)
				} else {
					count := int(cmd)
					for i := 0; i < count && y < streamH; i++ {
						var pix uint8
						if err := binary.Read(r, binary.LittleEndian, &pix); err != nil {
							break
						}

						drawY := (streamH - 1) - y
						targetIdx := (drawY * streamW) + x
						if targetIdx >= 0 && targetIdx < len(p.pixels) {
							p.pixels[targetIdx] = pix
						}
						y++
					}
				}
			}
		}
	}
	return nil
}

// NWXFrame represents a graphical frame containing rendering metadata for a specific frame in the NWX format.
// CellIndex specifies the logical index of the associated NWXCell in the CELT block.
// PhysicalIndex indicates the physical mapping index used in the CELT block for pixel data retrieval.
// Width defines the width dimension of the graphical frame in floating-point units.
// Height defines the height dimension of the graphical frame in floating-point units.
// Cell is a reference to the NWXCell containing pixel and size data for this frame.
type NWXFrame struct {
	CellIndex     uint32
	PhysicalIndex uint32
	Width         float32
	Height        float32
	Cell          *NWXCell
}

// NewNWXFrame creates and returns a new instance of NWXGraphicalFrame with default values.
func NewNWXFrame() *NWXFrame {
	return &NWXFrame{}
}

// Parse reads and extracts graphical frame data from the provided io.ReadSeeker stream. It returns the size of the parsed data and an error if any occurs.
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

	//if raw.CellIndex > 1000 {
	//	fmt.Println("HERE!!!!!")
	//}
	g.CellIndex = raw.CellIndex
	g.PhysicalIndex = raw.PhysicalIndex
	g.Width = raw.Width
	g.Height = raw.Height
	g.Cell = nil
	v := int(unsafe.Sizeof(raw))
	return v, nil
}

// NWXSequenceHeader represents the header of a NWX sequence with metadata about frames, scale, light, and sequences.
type NWXSequenceHeader struct {
	numFrames    uint32 // Questo ci servirà per mappare il blocco FRMT
	scaleX       uint32
	scaleY       uint32
	extraLight   uint32
	numSequences uint32 // Es: 32 azioni
}

// NewNWXSequenceHeader creates and returns a new instance of NWXSequenceHeader.
func NewNWXSequenceHeader() *NWXSequenceHeader {
	return &NWXSequenceHeader{}
}

// Parse reads and decodes sequence header data from the provided io.ReadSeeker into the NWXSequenceHeader struct.
func (s *NWXSequenceHeader) Parse(r io.ReadSeeker) error {
	var raw struct {
		Unknown1     uint32
		Unknown2     uint32
		NumFrames    uint32 // Questo ci servirà per mappare il blocco FRMT
		ScaleX       uint32
		ScaleY       uint32
		ExtraLight   uint32
		Pad          uint32
		NumSequences uint32 // Es: 32 azioni
	}
	if err := binary.Read(r, binary.LittleEndian, &raw); err != nil {
		return err
	}
	s.numFrames = raw.NumFrames
	s.scaleX = raw.ScaleX
	s.scaleY = raw.ScaleY
	s.extraLight = raw.ExtraLight
	s.numSequences = raw.NumSequences
	return nil

}

// NWX represents a collection of graphical frames and sequences used in a rendering system.
type NWX struct {
	//actions []*NWXSequence
	frames []*NWXFrame
}

// NewNWX initializes and returns a new instance of NWX.
func NewNWX() *NWX {
	return &NWX{}
}

// Parse reads and processes NWX file data from the provided reader, initializing frames, sequences, and cell mappings.
func (w *NWX) Parse(baseId string, r io.ReadSeeker) error {
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return err
	}
	var waxHeader struct {
		Signature  [4]byte
		Version    uint32
		Unknown    uint32
		ScaleX     float32
		ScaleY     float32
		CeltOffset uint32
		FrmtOffset uint32
		SeqOffset  uint32
	}

	if err := binary.Read(r, binary.LittleEndian, &waxHeader); err != nil {
		return err
	}
	if string(waxHeader.Signature[:]) != "WAXF" {
		return fmt.Errorf("invalid signature")
	}

	/*
		// Absolute chunk bases (skipping the 4 bytes of TAG and 4 bytes of SIZE = +8)
		seqBase := int64(waxHeader.SeqOffset) + 8

		// ==========================================
		// 1. SEQUENCE MAPPING (SEQT)
		// ==========================================
		if _, err := r.Seek(seqBase, io.SeekStart); err != nil {
			return fmt.Errorf("unable to seek to SEQT: %v", err)
		}

		seqHeader := NewNWXSequenceHeader()
		if err := seqHeader.Parse(r); err != nil {
			return err
		}

		// ==========================================
		// 2. FRAME MAPPING (FRMT)
		// ==========================================
		frmtBase := int64(waxHeader.FrmtOffset) + 8
		if _, err := r.Seek(frmtBase, io.SeekStart); err != nil {
			return fmt.Errorf("unable to seek to FRMT: %v", err)
		}

		// FrmtHeader must be 32byte
		var frmtHeader struct {
			ChunkSize uint32
			Pad1      uint32
			Pad2      uint32
			Pad3      uint32
			Width     float32
			Height    float32
			Pad6      uint32
			Pad7      uint32
		}
		if err := binary.Read(r, binary.LittleEndian, &frmtHeader); err != nil {
			return err
		}
		//if _, err := r.Seek(frmtBase+int64(unsafe.Sizeof(frmtHeader)), io.SeekStart); err != nil {
		//	return err
		//}
		if _, err := r.Seek(int64(waxHeader.CeltOffset), io.SeekStart); err != nil {
			return err
		}
		w.frames = make([]*NWXFrame, seqHeader.numFrames+1)
	*/
	if _, err := r.Seek(int64(waxHeader.CeltOffset), io.SeekStart); err != nil {
		return err
	}
	cellCache := make(map[uint32]*NWXCell)
	var vErr error
	for {
		idx, streamW, streamH, err := parseByMarkers(baseId, r)
		if err != nil {
			vErr = err
			//return fmt.Errorf("error reading cell %d in CELT: %v", i, err)
			break
		}
		f := NewNWXFrame()
		f.CellIndex = idx
		f.PhysicalIndex = idx
		f.Width = float32(streamW)
		f.Height = float32(streamH)
		w.frames = append(w.frames, f)
		//w.frames[idx] = f

		if cached, ok := cellCache[idx]; ok {
			f.Cell = cached
			continue
		}
		cell := NewNWXCell(fmt.Sprintf("phys_%d", f.PhysicalIndex))
		if err = cell.Parse(r, int(streamW), int(streamH), true); err != nil {
			return err
		}
		cellCache[f.CellIndex] = cell
		f.Cell = cell
	}
	/*
		w.frames = make([]*NWXFrame, seqHeader.numFrames+1)



		cellMax := uint32(0)
		for i := uint32(0); i < frmtHeader.ChunkSize; i++ {
			gFrame := NewNWXFrame()
			size, err := gFrame.Parse(r)
			if err != nil {
				return fmt.Errorf("error reading frame %d in FRMT: %v", i, err)
			}
			index := gFrame.CellIndex
			if int(index) >= len(w.frames) {
				return fmt.Errorf("cell %d not present in FRMT", index)
			}
			w.frames[index] = gFrame
			if index > cellMax {
				cellMax = index
			}
			i += uint32(size)
		}

		// ==========================================
		// 3. CELL AND PIXEL MAPPING (CELT)
		// ==========================================
		if _, err := r.Seek(int64(waxHeader.CeltOffset), io.SeekStart); err != nil {
			return err
		}
		cellCache := make(map[uint32]*NWXCell)
		for {
			idx, streamW, streamH, err := parseByMarkers(r)
			if err != nil {
				//return fmt.Errorf("error reading cell %d in CELT: %v", i, err)
				break
			}
			if idx > uint32(len(w.frames)) {
				fmt.Println("CELL NOT FOUND", idx, streamW, streamH)
				continue
			}
			f := w.frames[idx]
			if f == nil {
				fmt.Println("CELL NOT FOUND", idx, streamW, streamH)
				continue
			}
			if cached, ok := cellCache[f.PhysicalIndex]; ok {
				f.Cell = cached
				continue
			}
			cell := NewNWXCell(fmt.Sprintf("phys_%d", f.PhysicalIndex))
			if err = cell.Parse(r, int(streamW), int(streamH)); err != nil {
				return err
			}

			cellCache[f.PhysicalIndex] = cell
			f.Cell = cell
		}


	*/
	return vErr
}

var _streamCounter = uint32(0)

// parseByMarkers parses data from the provided io.ReadSeeker, locating markers to extract physIndex, streamW, and streamH.
func parseByMarkers(name string, r io.ReadSeeker) (uint32, uint32, uint32, error) {
	scanForMarker := func(r io.Reader, marker []byte) error {
		buf := make([]byte, 1)
		matchIdx := 0
		for matchIdx < len(marker) {
			if _, err := r.Read(buf); err != nil {
				return err
			}
			if buf[0] == marker[matchIdx] {
				matchIdx++
			} else {
				// Riavvolge il match se fallisce parzialmente
				if buf[0] == marker[0] {
					matchIdx = 1
				} else {
					matchIdx = 0
				}
			}
		}
		return nil
	}
	_streamCounter++

	//prologueMarker := []byte{0x4D, 0xFA, 0x6C, 0x00, 0xCD}
	dataMarker := []byte{0x5D, 0xFA, 0x6C, 0x00}
	//if err = scanForMarker(r, prologueMarker); err != nil {
	//	return 0, 0, 0, fmt.Errorf("prologo non trovato: %w", err)
	//}
	//if err = binary.Read(r, binary.LittleEndian, &physIndex); err != nil {
	//	return 0, 0, 0, fmt.Errorf("lettura physIndex fallita: %w", err)
	//}
	if err := scanForMarker(r, dataMarker); err != nil {
		return 0, 0, 0, fmt.Errorf("data marker non trovato per cella  %w", err)
	}

	//r.Seek(-20, io.SeekCurrent)
	//rav := make([]byte, 64)
	//binary.Read(r, binary.LittleEndian, &rav)
	//common.Hexdump(rav)
	//os.Exit(1)

	// Torniamo indietro di 12 byte: 4 (dataMarker) + 4 (streamH) + 4 (streamW) + 4 (Size) + 4 (Flags)
	if _, err := r.Seek(-32, io.SeekCurrent); err != nil {
		return 0, 0, 0, err
	}
	var streamW, streamH uint32
	var physIndex, size uint32
	var magic [4]byte
	var numCells, chunkSize uint32
	if err := binary.Read(r, binary.LittleEndian, &magic); err != nil {
		return 0, 0, 0, err
	}
	if err := binary.Read(r, binary.LittleEndian, &numCells); err != nil {
		return 0, 0, 0, err
	}
	if err := binary.Read(r, binary.LittleEndian, &chunkSize); err != nil {
		return 0, 0, 0, err
	}
	if err := binary.Read(r, binary.LittleEndian, &physIndex); err != nil {
		return 0, 0, 0, err
	}
	if err := binary.Read(r, binary.LittleEndian, &size); err != nil {
		return 0, 0, 0, err
	}
	if err := binary.Read(r, binary.LittleEndian, &streamW); err != nil {
		return 0, 0, 0, err
	}
	if err := binary.Read(r, binary.LittleEndian, &streamH); err != nil {
		return 0, 0, 0, err
	}

	//fmt.Printf("%s: magic %s numCells %d chunkSize %d physIndex %d size %d streamW %d streamH %d\n", name, string(magic[:]), numCells, chunkSize, physIndex, size, streamW, streamH)
	// 4. Ripristiniamo il cursore all'inizio della colTable
	// (saltiamo di nuovo i 4 byte del dataMarker)
	if _, err := r.Seek(4, io.SeekCurrent); err != nil {
		return 0, 0, 0, err
	}
	return physIndex, streamW, streamH, nil
}
