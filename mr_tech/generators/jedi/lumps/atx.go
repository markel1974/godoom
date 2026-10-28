package lumps

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ATX represents an animated texture definition from an Outlaws .ATX script.
type ATX struct {
	Name     string
	Rate     int
	Textures []string
	HasGoto  bool
	HasStop  bool
}

func NewATX(name string) *ATX {
	return &ATX{
		Name: name,
		Rate: 6,
	}
}

// Parse reads and parses the input data from the provided io.Reader into the ATX structure.
func (atx *ATX) Parse(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		switch strings.ToUpper(fields[0]) {
		case "RATE":
			if len(fields) > 1 {
				if r, err := strconv.Atoi(fields[1]); err == nil && r > 0 {
					atx.Rate = r
				}
			}
		case "TEXTURE":
			if len(fields) > 1 {
				tex := strings.TrimSpace(fields[1])
				atx.Textures = append(atx.Textures, tex)
			}
		case "GOTO":
			atx.HasGoto = true
		case "STOP":
			atx.HasStop = true
		case "GOTO_STOP":
			//TODO
		case "ATX":
			//TODO
		case "START_SOUND":
			//TODO
		case "INST":
			//TODO
		default:
			fmt.Println("Unknown ATX field:", fields[0])
		}
	}
	return scanner.Err()
}

// IsLooping returns true if this ATX represents a continuous repeating animation.
func (atx *ATX) IsLooping() bool {
	// Water, waterfall, lava, saw, lamp are continuous animations.
	// If it has no STOP, or if it loops and has multiple textures.
	if !atx.HasStop && len(atx.Textures) > 1 {
		return true
	}
	u := strings.ToUpper(atx.Name)
	if strings.Contains(u, "WATER") || strings.Contains(u, "POOL") || strings.Contains(u, "LAVA") || strings.Contains(u, "WFALL") {
		return len(atx.Textures) > 1
	}
	return atx.HasGoto && len(atx.Textures) > 1 && !atx.HasStop
}
