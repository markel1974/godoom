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

		// FIX 2: Calcolo della lunghezza esatta della colonna per evitare di leggere byte di padding
		colLength := streamH // Fallback per l'ultima colonna
		for nextX := x + 1; nextX < streamW; nextX++ {
			nextTarget := colTable[nextX] & 0x00FFFFFF
			if nextTarget != 0 {
				colLength = int(nextTarget - target)
				break
			}
		}

		if _, err := r.Seek(colTableBase+int64(target), io.SeekStart); err != nil {
			return err
		}

		if !isCompressed {
			// ==========================================
			// BRANCH RAW: Dati lineari (es. Barile)
			// ==========================================

			// Leggiamo solo i byte effettivi, ignorando eventuale padding a fine colonna
			pixelsToRead := streamH
			if colLength < streamH {
				pixelsToRead = colLength
			}

			for y := 0; y < pixelsToRead; y++ {
				var pix uint8
				if err := binary.Read(r, binary.LittleEndian, &pix); err != nil {
					break
				}

				// FIX 1: Trasparenza implicita (0) e Chroma Key LucasArts (255 = Magenta)
				if pix == 0 || pix == 255 {
					continue
				}

				//fmt.Println("current pix", pix)

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

						// Applichiamo il color key anche all'RLE per sicurezza
						if pix == 0 || pix == 255 {
							y++
							continue
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

	// ==========================================
	// 2. FRAME MAPPING (FRMT)
	// ==========================================
	frmtBase := int64(waxHeader.FrmtOffset) + 8
	if _, err := r.Seek(frmtBase, io.SeekStart); err != nil {
		return fmt.Errorf("unable to seek to FRMT: %v", err)
	}

	// FrmtHeader must be 32byte
	var frmtHeader struct {
		Flags     uint32
		InsertX   uint32
		InsertY   uint32
		Flip      uint32
		Width     float32
		Height    float32
		CellIndex uint32
		Pad7      uint32
	}
	if err := binary.Read(r, binary.LittleEndian, &frmtHeader); err != nil {
		return err
	}

	isCompressed := (frmtHeader.Flags & 0x1000) != 0
	//1. Bit 2 (0x0004): L'Interruttore Generale
	//2. Bit 12 (0x1000): Il Flag RLE
	//3. Bit 4 (0x0010) e Bit 5 (0x0020): Fullbright / Illuminazione
	//4. Bit 3 (0x0008) e Bit 6 (0x0040): Translucenza / Vetro
	//5. Bit 7, 8, 9, 10... (L'Incubo del Reverse Engineering): I Flag di FLIP X

	fmt.Printf("%s: %#X %d %d\n", baseId, frmtHeader.Flags, int(frmtHeader.Width), int(frmtHeader.Height))

	//fmt.Println(baseId, frmtHeader)

	if _, err := r.Seek(int64(waxHeader.CeltOffset), io.SeekStart); err != nil {
		return err
	}

	// 1. Legge l'Header Globale del blocco CELT (Solo 12 byte, una volta sola)
	var celtMagic, numCells, chunkSize uint32
	if err := binary.Read(r, binary.LittleEndian, &celtMagic); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &numCells); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &chunkSize); err != nil {
		return err
	}

	// Controllo di sicurezza
	if celtMagic != 1414284611 { // "CELT"
		return fmt.Errorf("non è un blocco CELT valido")
	}

	cellCache := make(map[uint32]*NWXCell)

	for i := uint32(0); i < numCells; i++ {
		// L'Header della Cella è di ESATTAMENTE 20 byte (5 uint32)
		var physIndex, size, streamW, streamH, magic uint32
		if err := binary.Read(r, binary.LittleEndian, &physIndex); err != nil {
			break
		}
		if err := binary.Read(r, binary.LittleEndian, &size); err != nil {
			break
		}
		if err := binary.Read(r, binary.LittleEndian, &streamW); err != nil {
			break
		}
		if err := binary.Read(r, binary.LittleEndian, &streamH); err != nil {
			break
		}
		if err := binary.Read(r, binary.LittleEndian, &magic); err != nil {
			break
		} // Consumiamo il Magic Number (0x006CFA5D) per completare l'header!

		f := NewNWXFrame()
		f.CellIndex = physIndex
		f.PhysicalIndex = physIndex
		f.Width = float32(streamW)
		f.Height = float32(streamH)
		w.frames = append(w.frames, f)

		// Ora siamo posizionati PERFETTAMENTE all'inizio della colTable (payload vero)
		payloadStart, _ := r.Seek(0, io.SeekCurrent)

		if cell := cellCache[physIndex]; cell != nil {
			f.Cell = cellCache[physIndex]
		} else {
			f.Cell = NewNWXCell(fmt.Sprintf("phys_%d", physIndex))
			if streamW > 0 && streamH > 0 {
				if err := f.Cell.Parse(r, int(streamW), int(streamH), isCompressed); err != nil {
					return err
				}
				cellCache[physIndex] = f.Cell
			}
		}
		if _, err := r.Seek(payloadStart+int64(size), io.SeekStart); err != nil {
			return err
		}
	}
	return nil
}
