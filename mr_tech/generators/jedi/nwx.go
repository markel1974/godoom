package jedi

import (
	"encoding/binary"
	"fmt"
	"io"
)

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

func (p *NWXCell) ParseStream(r io.ReadSeeker, width int, height int) error {
	p.sizeX, p.sizeY = width, height
	p.pixels = make([]byte, width*height)

	cellBase, _ := r.Seek(0, io.SeekCurrent)
	rawTable := make([]uint32, width)
	binary.Read(r, binary.LittleEndian, &rawTable)

	// Teniamo traccia della fine reale dei dati per il prossimo frame nello stream
	//furthestOffset := int64(width * 4)

	maxDataPos := int64(0)
	for x := 0; x < width; x++ {
		// Estrazione dell'offset: moltiplicatore 32 basato sui tuoi dump
		relOffset := int64(rawTable[x]>>24) * 16
		if relOffset == 0 {
			continue
		}

		r.Seek(cellBase+relOffset, io.SeekStart)

		y := 0
		for y < height {
			var cmd uint8
			if err := binary.Read(r, binary.LittleEndian, &cmd); err != nil {
				break
			}

			if cmd >= 128 {
				y += int(cmd - 128)
			} else if cmd > 0 {
				count := int(cmd)
				for i := 0; i < count && y < height; i++ {
					var pix uint8
					binary.Read(r, binary.LittleEndian, &pix)
					// MAPPA CORRETTAMENTE:
					// Se l'immagine è Column-Major, x è la coordinata orizzontale
					p.pixels[y*width+x] = pix
					y++
				}
			} else {
				break // End of Column
			}
		}

		curr, _ := r.Seek(0, io.SeekCurrent)
		if curr > maxDataPos {
			maxDataPos = curr
		}
	}

	// ALLINEAMENTO AL PROSSIMO BLOCCO (Fondamentale per lo streaming)
	// Se la cella successiva inizia a un offset allineato a 32 byte:
	// CAMBIO STRATEGIA:
	// Invece di allineare a 32, prova prima l'allineamento a 4 (DWORD)
	// o addirittura nessun allineamento se i dati sono densi.
	nextCellStart := (maxDataPos + 3) &^ 3

	// Debug: stampa l'offset di fine cella per capire se stiamo "mangiando" il file
	// fmt.Printf("Cella conclusa a %d, prossima attesa a %d\n", maxDataPos, nextCellStart)

	r.Seek(nextCellStart, io.SeekStart)

	return nil
}

// NWXGraphicalFrame represents a graphical frame with positional and dimensional attributes used in NWX structures.
type NWXGraphicalFrame struct {
	CellIndex uint32
	Cell      *NWXCell
	Width     float32
	Height    float32
}

// NewNWXGraphicalFrame creates and returns a new instance of NWXGraphicalFrame with default values.
func NewNWXGraphicalFrame() *NWXGraphicalFrame {
	return &NWXGraphicalFrame{}
}

// Parse reads and decodes binary data from the provided io.ReadSeeker into the NWXGraphicalFrame structure.
func (g *NWXGraphicalFrame) Parse(r io.ReadSeeker) error {
	var raw struct {
		InsertX   int32  // Offset X per centrare l'immagine
		InsertY   int32  // Offset Y per centrare l'immagine
		Flags     uint32 // Flip orizzontale/verticale o attributi
		CellIndex uint32 // ID della cella nel blocco CELT (i pixel reali)
		Pad1      uint32
		Pad2      uint32
		Width     float32 // Dimensione Float (Fixed Point tradotto) o Bounding Box
		Height    float32 // Dimensione Float
		Pad3      uint32
		Pad4      uint32
	}

	//raw := make([]byte, 32)
	if err := binary.Read(r, binary.LittleEndian, &raw); err != nil {
		return err
	}
	//g.InsertX = raw.InsertX
	//g.InsertY = raw.InsertY
	//g.CellOffset = raw.CellOffset
	//fmt.Println("--------------- CELL -------------------")
	//common.Hexdump(raw)
	//fmt.Println("CELL", raw)
	g.CellIndex = raw.CellIndex
	g.Width = raw.Width
	g.Height = raw.Height
	g.Cell = nil
	return nil
}

// NWXFrame represents a single frame in an animation sequence using an index and associated flags.
type NWXFrame struct {
	Index uint16
	Flags uint16
}

// NewNWXAnimCommand creates a new NWXFrame with the given frame index and flags.
func NewNWXAnimCommand(frameIdx uint16, flags uint16) *NWXFrame {
	return &NWXFrame{
		Index: frameIdx,
		Flags: flags,
	}
}

// NWXSequence represents a sequence of animation frames within an NWX structure.
type NWXSequence struct {
	frames []*NWXFrame
}

// NewNWXSequence creates and returns a new instance of NWXSequence.
func NewNWXSequence() *NWXSequence {
	return &NWXSequence{}
}

// Parse reads a sequence of animation commands from the provided io.ReadSeeker and populates the frames in the NWXSequence.
func (wa *NWXSequence) Parse(r io.ReadSeeker) error {
	for {
		var cmd uint32
		if err := binary.Read(r, binary.LittleEndian, &cmd); err != nil {
			return err //fmt.Errorf("error reading sequence %d: %v", actIdx, err)
		}
		frameIdx := uint16(cmd & 0xFFFF)
		flags := uint16(cmd >> 16)
		// 0xFFFF è il terminatore universale di sequenza
		if frameIdx == 0xFFFF {
			break
		}
		c := NewNWXAnimCommand(frameIdx, flags)
		wa.frames = append(wa.frames, c)
	}

	return nil
}

// GetViews returns an array of pointers to NWXView objects held within the NWXActions instance.
//func (wa *NWXActions) GetViews() []*NWXView { return wa.views }

// NWXSequenceHeader represents the header structure for an NWX sequence, providing metadata such as frame count and scaling.
type NWXSequenceHeader struct {
	numFrames    uint32 // Questo ci servirà per mappare il blocco FRMT
	scaleX       uint32
	scaleY       uint32
	extraLight   uint32
	numSequences uint32 // Es: 32 azioni
}

func NewNWXSequenceHeader() *NWXSequenceHeader {
	return &NWXSequenceHeader{}
}

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

// Struttura per immagazzinare la singola istruzione

// NWX represents a data structure containing animations and graphical frames for rendering sequences in NWX format.
type NWX struct {
	actions []*NWXSequence
	frames  []*NWXGraphicalFrame
}

// NewNWX initializes and returns a new instance of the NWX struct.
func NewNWX() *NWX {
	return &NWX{}
}

func (w *NWX) Parse(baseId string, r io.ReadSeeker) error {
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return err
	}

	// --- 0. HEADER WAXF ---
	var waxf struct {
		Signature  [4]byte
		Version    uint32
		Unknown    uint32
		ScaleX     float32
		ScaleY     float32
		CeltOffset uint32
		FrmtOffset uint32
		SeqOffset  uint32
	}

	if err := binary.Read(r, binary.LittleEndian, &waxf); err != nil {
		return err
	}

	if string(waxf.Signature[:]) != "WAXF" {
		return fmt.Errorf("invalid signature")
	}

	// Basi assolute dei chunk (saltando i 4 byte del TAG e i 4 byte della SIZE = +8)
	celtBase := int64(waxf.CeltOffset) + 8
	frmtBase := int64(waxf.FrmtOffset) + 8
	seqBase := int64(waxf.SeqOffset) + 8

	// ==========================================
	// 1. MAPPATURA SEQUENZE (SEQT)
	// ==========================================
	if _, err := r.Seek(seqBase, io.SeekStart); err != nil {
		return fmt.Errorf("impossibile posizionarsi su SEQT: %v", err)
	}

	seqHeader := NewNWXSequenceHeader()
	if err := seqHeader.Parse(r); err != nil {
		return err
	}

	fmt.Printf("\n--- SEQT Mappato (all'offset %d) ---\n", seqBase)
	fmt.Printf("NumFrames globale: %d | NumSequences: %d\n", seqHeader.numFrames, seqHeader.numSequences)

	w.actions = make([]*NWXSequence, seqHeader.numSequences)

	// ATTENZIONE: Nessuna lettura di "seqEntries" a 32-bit qui!
	// Subito dopo i 32 byte dell'header (NWXSequenceHeader), inizia immediatamente
	// lo stream del bytecode delle animazioni.
	for actIdx := uint32(0); actIdx < seqHeader.numSequences; actIdx++ {
		sequence := NewNWXSequence()
		if err := sequence.Parse(r); err != nil {
			return fmt.Errorf("errore parsing bytecode sequence %d: %v", actIdx, err)
		}
		w.actions[actIdx] = sequence
	}

	// ==========================================
	// 2. MAPPATURA FRAMES (FRMT)
	// ==========================================
	if _, err := r.Seek(frmtBase, io.SeekStart); err != nil {
		return fmt.Errorf("impossibile posizionarsi su FRMT: %v", err)
	}

	// Il primo uint32 di FRMT è la dimensione in byte dell'intero blocco dati
	var frmtChunkSize uint32
	if err := binary.Read(r, binary.LittleEndian, &frmtChunkSize); err != nil {
		return err
	}

	// Saltiamo l'intero header di 32 byte per posizionarci esattamente all'inizio del primo Frame
	if _, err := r.Seek(frmtBase+32, io.SeekStart); err != nil {
		return err
	}

	// La dimensione della tua struct NWXGraphicalFrame è graniticamente 40 byte.
	// Calcoliamo i frame reali (saranno circa 91, non 412).
	realNumFrames := (frmtChunkSize - 32) / 40
	w.frames = make([]*NWXGraphicalFrame, realNumFrames)

	celtNumCells := uint32(0)
	for i := uint32(0); i < realNumFrames; i++ {
		gFrame := NewNWXGraphicalFrame()
		if err := gFrame.Parse(r); err != nil {
			return fmt.Errorf("errore lettura frame %d in FRMT: %v", i, err)
		}
		w.frames[i] = gFrame
		if gFrame.CellIndex > celtNumCells {
			celtNumCells = gFrame.CellIndex
		}
	}

	fmt.Printf("\n--- FRMT Mappato ---\nLetti %d frame esatti da 40 byte.\n", len(w.frames))

	frames := make([]*NWXGraphicalFrame, celtNumCells+1)
	for _, f := range w.frames {
		frames[f.CellIndex] = f
	}
	// ==========================================
	// 3. MAPPATURA CELLE E PIXEL (CELT)
	// ==========================================
	streamStart := celtBase + 88
	if _, err := r.Seek(streamStart, io.SeekStart); err != nil {
		return err
	}
	for _, frame := range frames {
		if frame == nil {
			continue
		}
		cellId := fmt.Sprintf("%s_cell_%d", baseId, frame.CellIndex)
		cell := NewNWXCell(cellId, nil)
		if err := cell.ParseStream(r, int(frame.Width), int(frame.Height)); err != nil {
			return fmt.Errorf("errore lettura cella %d in CELT: %v", frame.CellIndex, err)
		}
		frame.Cell = cell
	}

	fmt.Printf("\n--- CELT Estratto ---\nElaborato lo stream lineare di celle.\n")
	return nil
}
