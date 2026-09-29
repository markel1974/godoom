package lumps

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

// Anim3 defines an animation with properties for frames, looping, playback speed, and an optional descriptive name.
type Anim3 struct {
	FirstFrame int
	NumFrames  int
	LoopFrames int
	FPS        int
	Name       string
}

// AnimConfig3 represents the configuration for managing a set of animations and their associated leg offsets.
type AnimConfig3 struct {
	Animations []Anim3
	LegsOffset int
}

// NewAnimConfig3 reads animation configuration data from an io.Reader and returns an AnimConfig3 structure or an error.
func NewAnimConfig3(rs io.Reader) (*AnimConfig3, error) {
	config := &AnimConfig3{
		Animations: make([]Anim3, 0, 25),
	}

	scanner := bufio.NewScanner(rs)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Remove comments at the end of the line if they don't start the line
		commentIdx := strings.Index(line, "//")
		var name string
		if commentIdx != -1 {
			name = strings.TrimSpace(line[commentIdx+2:])
			line = strings.TrimSpace(line[:commentIdx])
		}

		if len(line) == 0 {
			continue
		}

		// Skip non-numeric lines like "sex m" or "footsteps normal"
		if line[0] < '0' || line[0] > '9' {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 4 {
			first, _ := strconv.Atoi(fields[0])
			num, _ := strconv.Atoi(fields[1])
			loop, _ := strconv.Atoi(fields[2])
			fps, _ := strconv.Atoi(fields[3])

			if len(name) == 0 {
				name = "anim_" + strconv.Itoa(len(config.Animations))
			}

			config.Animations = append(config.Animations, Anim3{
				FirstFrame: first,
				NumFrames:  num,
				LoopFrames: loop,
				FPS:        fps,
				Name:       name,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Calculate legsOffset: number of frames used by TORSO animations
	// Typically TORSO starts at index 6 and LEGS starts at index 13
	torsoStart := -1
	legsStart := -1

	for _, anim := range config.Animations {
		if strings.HasPrefix(anim.Name, "TORSO_") && torsoStart == -1 {
			torsoStart = anim.FirstFrame
		}
		if strings.HasPrefix(anim.Name, "LEGS_") && legsStart == -1 {
			legsStart = anim.FirstFrame
		}
	}

	if torsoStart != -1 && legsStart != -1 {
		config.LegsOffset = legsStart - torsoStart
	}

	return config, nil
}
