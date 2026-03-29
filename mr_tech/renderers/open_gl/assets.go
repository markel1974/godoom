package open_gl

import (
	"embed"
	"io/fs"
	"os"
)

// assets represents an embedded file system containing application resources such as container or assets.
//
//go:embed assets
var assets embed.FS

// Assets is a struct utilized for handling file operations within an embedded assets file system.
type Assets struct {
}

// BasePath constructs a platform-specific file path by combining the "assets" directory with the given relative path.
func (w *Assets) BasePath(vPath string) string {
	return "assets" + string(os.PathSeparator) + vPath
}

// Read retrieves the contents of the specified file path `p` from the embedded file system and returns it as a byte slice.
func (w *Assets) Read(p string) ([]byte, error) {
	data, err := fs.ReadFile(assets, w.BasePath(p))
	if err != nil {
		return nil, err
	}
	return data, nil
}

// ReadMulti reads two files specified by their paths and returns their contents as byte slices or an error if any occurs.
func (w *Assets) ReadMulti(a string, b string) ([]byte, []byte, error) {
	aOut, err := w.Read(a)
	if err != nil {
		return nil, nil, err
	}
	bOut, err := w.Read(b)
	if err != nil {
		return nil, nil, err
	}
	return aOut, bOut, nil
}
