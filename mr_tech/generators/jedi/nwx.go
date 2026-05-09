package jedi

import (
	"encoding/binary"
	"fmt"
	"io"
	"unsafe"
)

// NWXCell represents a 2D grid cell with unique identifier, dimensions, and pixel data.
type NWXCell struct {
	id     string
	sizeX  int
	sizeY  int
	pixels []byte
}

// GetId returns the unique identifier of the NWXCell as a string.
func (p *NWXCell) GetId() string { return p.id }

// GetSize returns the width and height of the NWXCell as two integers.
func (p *NWXCell) GetSize() (int, int) { return p.sizeX, p.sizeY }

// GetPixels retrieves the pixel data of the NWXCell as a byte slice.
func (p *NWXCell) GetPixels() []byte { return p.pixels }

// NewNWXCell creates and initializes a new NWXCell instance with the specified identifier.
func NewNWXCell(id string) *NWXCell {
	return &NWXCell{id: id}
}

// Parse decodes a stream of pixel data into the NWXCell using the provided stream dimensions and color table offset.
func (p *NWXCell) Parse(r io.ReadSeeker, colTableBase int64, streamW, streamH, streamSize int) error {
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

		currentOffset := colTableBase + int64(target)
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

// NWXFrame represents a single frame entity, defining its dimensions and associated cell data in the NWX system.
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

// Parse reads and parses binary data from the provided io.ReadSeeker and populates the NWXFrame fields with the extracted values.
// It returns the size of the parsed data structure and an error if the read operation fails.
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

// NWXSequenceHeader represents the metadata for a sequence, including frame count, scale, lighting, and sequence count.
type NWXSequenceHeader struct {
	numFrames    uint32 // Questo ci servirà per mappare il blocco FRMT
	scaleX       uint32
	scaleY       uint32
	extraLight   uint32
	numSequences uint32 // Es: 32 azioni
}

// NewNWXSequenceHeader initializes and returns a pointer to a new NWXSequenceHeader instance.
func NewNWXSequenceHeader() *NWXSequenceHeader {
	return &NWXSequenceHeader{}
}

// Parse reads and populates the NWXSequenceHeader fields from the provided io.ReadSeeker using binary little-endian format.
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

// NWX represents a structure for managing collections of NWXFrame objects.
type NWX struct {
	//actions []*NWXSequence
	frames []*NWXFrame
}

// NewNWX creates and returns a new instance of the NWX struct.
func NewNWX() *NWX {
	return &NWX{}
}

// Parse reads and parses NWX data from the given io.ReadSeeker and initializes internal structures with frame and cell data.
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

	//isCompressed := (frmtHeader.Flags & 0x1000) != 0
	//isCompressed = false
	//1. Bit 2 (0x0004): L'Interruttore Generale
	//2. Bit 12 (0x1000): Il Flag RLE
	//3. Bit 4 (0x0010) e Bit 5 (0x0020): Fullbright / Illuminazione
	//4. Bit 3 (0x0008) e Bit 6 (0x0040): Translucenza / Vetro
	//5. Bit 7, 8, 9, 10... (L'Incubo del Reverse Engineering): I Flag di FLIP X

	//fmt.Printf("%s: %#X %d %d\n", baseId, frmtHeader.Flags, int(frmtHeader.Width), int(frmtHeader.Height))

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
				if err := f.Cell.Parse(r, payloadStart, int(streamW), int(streamH), int(size)); err != nil {
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
