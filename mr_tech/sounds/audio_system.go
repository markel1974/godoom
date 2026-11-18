package sounds

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/ebitengine/oto/v3"
)

/*
const (
	TargetSampleRate = 44100
	TargetChannels   = 2
)

*/

// AudioSystem gestisce l'output verso Oto con canali separati per Musica ed Effetti
type AudioSystem struct {
	ctx         *oto.Context
	musicPlayer *oto.Player
	sfxPlayers  []*oto.Player
}

// NewAudioSystem inizializza l'unico contesto globale di Oto (da chiamare all'avvio)
func NewAudioSystem() (*AudioSystem, error) {
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

	return &AudioSystem{
		ctx: ctx,
	}, nil
}

// PrepareWav prende il []byte WAV prodotto da un generatore (Doom, Q1, Q2, Q3, D3)
// e lo converte in PCM pronto per essere suonato istantaneamente senza latenza.
func PrepareWav(wavBytes []byte) ([]byte, error) {
	if len(wavBytes) < 44 || string(wavBytes[0:4]) != "RIFF" || string(wavBytes[8:12]) != "WAVE" {
		return nil, errors.New("dati non validi: atteso header RIFF/WAVE")
	}

	var srcRate, srcChannels, srcBits int
	var rawPCM []byte
	offset := 12

	// Legge dinamicamente i chunk "fmt " e "data" ignorando metadati extra (es. loop/cue)
	for offset+8 <= len(wavBytes) {
		chunkID := string(wavBytes[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(wavBytes[offset+4 : offset+8]))
		offset += 8

		if offset+chunkSize > len(wavBytes) {
			break
		}

		switch chunkID {
		case "fmt ":
			if chunkSize >= 16 {
				audioFormat := binary.LittleEndian.Uint16(wavBytes[offset : offset+2])
				if audioFormat != 1 {
					return nil, fmt.Errorf("formato WAV %d non supportato (richiesto PCM = 1)", audioFormat)
				}
				srcChannels = int(binary.LittleEndian.Uint16(wavBytes[offset+2 : offset+4]))
				srcRate = int(binary.LittleEndian.Uint32(wavBytes[offset+4 : offset+8]))
				srcBits = int(binary.LittleEndian.Uint16(wavBytes[offset+14 : offset+16]))
			}
		case "data":
			rawPCM = wavBytes[offset : offset+chunkSize]
		}

		offset += chunkSize
		if chunkSize%2 != 0 {
			offset++
		}
	}

	if srcRate == 0 || len(rawPCM) == 0 {
		return nil, errors.New("chunk fmt o data mancanti nel WAV")
	}

	return resampleToTarget(rawPCM, srcRate, srcChannels, srcBits)
}

// PlayMusicWav accetta un WAV dal generatore musica e lo avvia in loop continuo
func (a *AudioSystem) PlayMusicWav(wavBytes []byte, volume float64) error {
	pcm, err := PrepareWav(wavBytes)
	if err != nil {
		return err
	}
	a.PlayMusicPCM(pcm, volume)
	return nil
}

// PlayMusicPCM avvia una traccia già convertita con PrepareWav
func (a *AudioSystem) PlayMusicPCM(pcm []byte, volume float64) {
	a.StopMusic()
	a.musicPlayer = a.ctx.NewPlayer(&loopReader{data: pcm})
	a.musicPlayer.SetVolume(volume)
	a.musicPlayer.Play()
}

// SetMusicVolume cambia il volume della musica in tempo reale (0.0 - 1.0)
func (a *AudioSystem) SetMusicVolume(volume float64) {
	if a.musicPlayer != nil {
		a.musicPlayer.SetVolume(volume)
	}
}

// StopMusic ferma e rilascia il player della musica
func (a *AudioSystem) StopMusic() {
	if a.musicPlayer != nil {
		_ = a.musicPlayer.Close()
		a.musicPlayer = nil
	}
}

// PlaySFXWav accetta un WAV dal generatore effetti e lo riproduce subito
func (a *AudioSystem) PlaySFXWav(wavBytes []byte, volume float64) error {
	pcm, err := PrepareWav(wavBytes)
	if err != nil {
		return err
	}
	a.PlaySFXPCM(pcm, volume)
	return nil
}

// PlaySFXPCM riproduce un effetto già convertito con PrepareWav (consigliato per suoni frequenti)
func (a *AudioSystem) PlaySFXPCM(pcm []byte, volume float64) {
	a.cleanupSFX()
	p := a.ctx.NewPlayer(bytes.NewReader(pcm))
	p.SetVolume(volume)
	p.Play()
	a.sfxPlayers = append(a.sfxPlayers, p)
}

func (a *AudioSystem) cleanupSFX() {
	active := a.sfxPlayers[:0]
	for _, p := range a.sfxPlayers {
		if p.IsPlaying() {
			active = append(active, p)
		} else {
			_ = p.Close()
		}
	}
	a.sfxPlayers = active
}

// resampleToTarget uniforma 8-bit/16-bit, mono/stereo e qualsiasi frequenza a 44100Hz 16-bit Stereo
func resampleToTarget(rawPCM []byte, srcRate, srcChannels, srcBits int) ([]byte, error) {
	if srcBits != 8 && srcBits != 16 {
		return nil, fmt.Errorf("profondità bit non supportata: %d", srcBits)
	}
	if srcChannels != 1 && srcChannels != 2 {
		return nil, fmt.Errorf("numero canali non supportato: %d", srcChannels)
	}

	bytesPerSrcFrame := (srcBits / 8) * srcChannels
	srcFrames := len(rawPCM) / bytesPerSrcFrame
	dstFrames := int(float64(srcFrames) * float64(TargetSampleRate) / float64(srcRate))

	out := make([]byte, dstFrames*4)

	for i := 0; i < dstFrames; i++ {
		srcIdx := int(float64(i) * float64(srcRate) / float64(TargetSampleRate))
		if srcIdx >= srcFrames {
			srcIdx = srcFrames - 1
		}
		frameOffset := srcIdx * bytesPerSrcFrame

		var left, right int16
		if srcBits == 8 {
			l := (int16(rawPCM[frameOffset]) - 128) << 8
			left = l
			if srcChannels == 2 {
				right = (int16(rawPCM[frameOffset+1]) - 128) << 8
			} else {
				right = l
			}
		} else {
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

type loopReader struct {
	data []byte
	pos  int
}

func (r *loopReader) Read(p []byte) (int, error) {
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
