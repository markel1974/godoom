package proc

import (
	"io"
	"regexp"
	"strings"
)

var guiBgRegex = regexp.MustCompile(`(?i)background\s+["']?([^"';\s}]+)["']?`)

var guiIgnoreKeywords = []string{
	"dirt", "scanlines", "mask", "static", "reflect", "outerglow",
	"outershadow", "addhighlight", "scratch", "cursor", "pointer",
}

// ExtractGuiBackground reads a Doom 3 .gui file and returns the primary background image/material path.
func ExtractGuiBackground(arc IPkArchive, guiPath string) string {
	if arc == nil || len(guiPath) == 0 {
		return ""
	}
	clean := strings.TrimSpace(strings.ReplaceAll(guiPath, "\\", "/"))
	clean = strings.TrimPrefix(clean, "/")
	if !strings.HasSuffix(strings.ToLower(clean), ".gui") {
		clean += ".gui"
	}

	f, err := arc.Open(clean)
	if err != nil {
		return ""
	}
	data, err := io.ReadAll(f)
	if err != nil || len(data) == 0 {
		return ""
	}

	matches := guiBgRegex.FindAllStringSubmatch(string(data), -1)
	var firstCandidate string
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		bg := strings.TrimSpace(m[1])
		if len(bg) == 0 {
			continue
		}
		bgLower := strings.ToLower(bg)
		ignored := false
		for _, kw := range guiIgnoreKeywords {
			if strings.Contains(bgLower, kw) {
				ignored = true
				break
			}
		}
		if !ignored {
			return bg
		}
		if firstCandidate == "" {
			firstCandidate = bg
		}
	}
	return firstCandidate
}
