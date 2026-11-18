package sounds

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"

	"github.com/ebitengine/oto/v3"
)

const (
	TargetSampleRate = 44100
	TargetChannels   = 2
)

// SoundSystem gestisce separatamente il canale musica e i canali effetti
type SoundSystem struct {
	ctx         *oto.Context
	musicPlayer *oto.Player
	sfxPlayers  []*oto.Player // Tiene traccia degli effetti attivi per pulirli
}

// NewSoundSystem inizializza l'unico contesto audio globale
func NewSoundSystem() (*SoundSystem, error) {
	op := &oto.NewContextOptions{
		SampleRate:   TargetSampleRate,
		ChannelCount: TargetChannels,
		Format:       oto.FormatSignedInt16LE,
	}

	ctx, readyChan, err := oto.NewContext(op)
	if err != nil {
		return nil, err
	}
	<-readyChan

	return &SoundSystem{
		ctx: ctx,
	}, nil
}

// PlayMusic avvia una traccia musicale in loop con volume dedicato
func (s *SoundSystem) PlayMusic(wavBytes []byte, volume float64) error {
	// Ferma eventuale musica precedente
	if s.musicPlayer != nil {
		_ = s.musicPlayer.Close()
	}

	pcm, err := decodeAndResampleToTarget(wavBytes)
	if err != nil {
		return err
	}

	// Avvolge i byte in un reader infinito per il loop continuo
	loopReader := &infiniteLoopReader{data: pcm}

	s.musicPlayer = s.ctx.NewPlayer(loopReader)
	s.musicPlayer.SetVolume(volume) // Valore tra 0.0 e 1.0
	s.musicPlayer.Play()
	return nil
}

// PlaySFX riproduce un effetto sonoro sovrapponendolo alla musica e agli altri effetti
func (s *SoundSystem) PlaySFX(wavBytes []byte, volume float64) error {
	// Pulisce i player degli effetti già terminati per liberare memoria
	s.cleanupFinishedSFX()

	pcm, err := decodeAndResampleToTarget(wavBytes)
	if err != nil {
		return err
	}

	sfxPlayer := s.ctx.NewPlayer(bytes.NewReader(pcm))
	sfxPlayer.SetVolume(volume)
	sfxPlayer.Play()

	s.sfxPlayers = append(s.sfxPlayers, sfxPlayer)
	return nil
}

func (s *SoundSystem) cleanupFinishedSFX() {
	active := s.sfxPlayers[:0]
	for _, p := range s.sfxPlayers {
		if p.IsPlaying() {
			active = append(active, p)
		} else {
			_ = p.Close()
		}
	}
	s.sfxPlayers = active
}

// infiniteLoopReader permette alla musica di ricominciare da capo senza interruzioni
type infiniteLoopReader struct {
	data []byte
	pos  int
}

func (r *infiniteLoopReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	written := 0
	for written < len(p) {
		n := copy(p[written:], r.data[r.pos:])
		written += n
		r.pos += n
		if r.pos >= len(r.data) {
			r.pos = 0
		}
	}
	return written, nil
}

// decodeAndResampleToTarget legge il WAV e lo converte nel formato del contesto (44100Hz, 16-bit Stereo)
func decodeAndResampleToTarget(wavBytes []byte) ([]byte, error) {
	if len(wavBytes) < 44 || string(wavBytes[0:4]) != "RIFF" {
		return nil, errors.New("file WAV non valido")
	}

	var srcRate, srcChannels, srcBits int
	var rawPCM []byte
	offset := 12

	for offset+8 <= len(wavBytes) {
		id := string(wavBytes[offset : offset+4])
		size := int(binary.LittleEndian.Uint32(wavBytes[offset+4 : offset+8]))
		offset += 8
		if offset+size > len(wavBytes) {
			break
		}
		if id == "fmt " && size >= 16 {
			srcChannels = int(binary.LittleEndian.Uint16(wavBytes[offset+2 : offset+4]))
			srcRate = int(binary.LittleEndian.Uint32(wavBytes[offset+4 : offset+8]))
			srcBits = int(binary.LittleEndian.Uint16(wavBytes[offset+14 : offset+16]))
		} else if id == "data" {
			rawPCM = wavBytes[offset : offset+size]
		}
		offset += size
		if size%2 != 0 {
			offset++
		}
	}

	if srcRate == 0 || len(rawPCM) == 0 {
		return nil, errors.New("dati audio mancanti")
	}

	// Conversione in campioni normalizzati int16 mono/stereo
	bytesPerSrcFrame := (srcBits / 8) * srcChannels
	srcFrames := len(rawPCM) / bytesPerSrcFrame
	dstFrames := int(float64(srcFrames) * float64(TargetSampleRate) / float64(srcRate))

	out := make([]byte, dstFrames*4) // 4 byte per frame (16-bit * 2 canali)

	for i := range dstFrames {
		// Ricampionamento lineare semplice (nearest-neighbor)
		srcIdx := int(float64(i) * float64(srcRate) / float64(TargetSampleRate))
		if srcIdx >= srcFrames {
			srcIdx = srcFrames - 1
		}
		frameOffset := srcIdx * bytesPerSrcFrame

		var left, right int16
		if srcBits == 8 {
			// Quake 1: 8-bit unsigned (0..255, silenzio a 128) -> 16-bit signed
			l := (int16(rawPCM[frameOffset]) - 128) << 8
			left = l
			if srcChannels == 2 {
				right = (int16(rawPCM[frameOffset+1]) - 128) << 8
			} else {
				right = l
			}
		} else if srcBits == 16 {
			left = int16(binary.LittleEndian.Uint16(rawPCM[frameOffset : frameOffset+2]))
			if srcChannels == 2 {
				right = int16(binary.LittleEndian.Uint16(rawPCM[frameOffset+2 : frameOffset+4]))
			} else {
				right = left
			}
		}

		outOffset := i * 4
		binary.LittleEndian.PutUint16(out[outOffset:outOffset+2], uint16(left))
		binary.LittleEndian.PutUint16(out[outOffset+2:outOffset+4], uint16(right))
	}

	return out, nil
}

/*
// Esempio d'uso
func main() {
	sys, err := NewSoundSystem()
	if err != nil {
		panic(err)
	}

	// Simulazione: avvia musica al 40% di volume ed effetti all'80%
	// sys.PlayMusic(musicWavBytes, 0.4)
	// sys.PlaySFX(shotgunWavBytes, 0.8)

	time.Sleep(5 * time.Second)
}


*/
