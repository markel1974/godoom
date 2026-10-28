package lumps

import (
	"bufio"
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

// ParseATX parses an ATX file from an io.Reader.
func ParseATX(name string, r io.Reader) (*ATX, error) {
	atx := &ATX{
		Name: name,
		Rate: 6,
	}
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
		}
	}
	return atx, scanner.Err()
}

// IsLooping returns true if this ATX represents a continuous repeating animation.
func (a *ATX) IsLooping() bool {
	// Water, waterfall, lava, saw, lamp are continuous animations.
	// If it has no STOP, or if it loops and has multiple textures.
	if !a.HasStop && len(a.Textures) > 1 {
		return true
	}
	u := strings.ToUpper(a.Name)
	if strings.Contains(u, "WATER") || strings.Contains(u, "POOL") || strings.Contains(u, "LAVA") || strings.Contains(u, "WFALL") {
		return len(a.Textures) > 1
	}
	return a.HasGoto && len(a.Textures) > 1 && !a.HasStop
}
