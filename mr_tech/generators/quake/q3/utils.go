package q3

import (
	"path/filepath"
)

func BaseName(in string) string {
	p := filepath.Ext(in)
	if len(p) == 0 {
		return in
	}
	return in[:len(in)-len(p)]
}
