package lumps

import (
	"bufio"
	"io"
	"strings"
)

// Skin represents a wrapper around an io.Reader, typically used for skin-related data parsing tasks.
type Skin struct {
	rs io.Reader
}

// NewSkin initializes and returns a new Skin instance with the provided io.Reader as its data source.
func NewSkin(rs io.Reader) *Skin {
	return &Skin{
		rs: rs,
	}
}

// Parse reads and parses data from the provided io.Reader, returning a map of mesh names to texture paths.
func (s *Skin) Parse() (map[string]string, error) {
	skinMap := make(map[string]string)
	scanner := bufio.NewScanner(s.rs)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			continue
		}
		// A line looks like: meshName,texturePath
		parts := strings.Split(line, ",")
		if len(parts) >= 2 {
			meshName := strings.TrimSpace(parts[0])
			texPath := strings.TrimSpace(parts[1])

			// Some tags (like tag_head,) have empty texture paths, we can ignore or store them
			if len(texPath) > 0 {
				skinMap[meshName] = texPath
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return skinMap, nil
}
