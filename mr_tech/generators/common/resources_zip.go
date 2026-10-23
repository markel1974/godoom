package common

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
)

// ZipDirEntry represents a directory entry within a ZIP archive, providing methods to retrieve its attributes.
type ZipDirEntry struct {
	file *zip.File
}

// Name returns the base name of the file, excluding the directory path and any trailing slash.
func (e ZipDirEntry) Name() string {
	return path.Base(strings.TrimSuffix(path.Clean(e.file.Name), "/"))
}

// IsDir reports whether the zipDirEntry represents a directory based on the trailing slash in its file name.
func (e ZipDirEntry) IsDir() bool {
	return strings.HasSuffix(e.file.Name, "/")
}

// Type returns the file mode of the entry, indicating if it is a directory or a regular file.
func (e ZipDirEntry) Type() fs.FileMode {
	if e.IsDir() {
		return fs.ModeDir
	}
	return 0
}

// Info returns the FileInfo of the zip entry and an error, if any occurs during retrieval.
func (e ZipDirEntry) Info() (fs.FileInfo, error) {
	return e.file.FileInfo(), nil
}

// ResourcesZip is a type for interacting with ZIP files, enabling setup, file extraction, and directory reading.
type ResourcesZip struct {
	z *zip.ReadCloser
}

// Setup initializes the ReaderZip by opening the zip file at the given path. It returns an error if the file cannot be opened.
func (z *ResourcesZip) Setup(p string) error {
	zf, err := zip.OpenReader(p)
	if err != nil {
		return err
	}
	z.z = zf
	return nil
}

// Open extracts a file from the zip archive by name and returns a reader to access its content.
func (z *ResourcesZip) Open(in string) (IReader, error) {
	f, err := z.z.Open(in)
	if err != nil {
		return nil, fmt.Errorf("open %q in zip: %w", in, err)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("read %q from zip: %w", in, err)
	}
	return NewReaderMem(data), nil
}

// ReadDir reads the contents of the specified directory within the zip archive and returns a slice of directory entries.
func (z *ResourcesZip) ReadDir(fPath string) ([]os.DirEntry, error) {
	safeZipPath := func(p string) (string, error) {
		p = strings.ReplaceAll(p, "\\", "/")
		p = path.Clean(p)
		if strings.HasPrefix(p, "/") || p == ".." || strings.HasPrefix(p, "../") {
			return "", fmt.Errorf("unsafe zip path: %q", p)
		}
		return p, nil
	}
	dir, dErr := safeZipPath(fPath)
	if dErr != nil {
		return nil, dErr
	}
	if dir == "." {
		dir = ""
	} else {
		dir += "/"
	}
	entries := map[string]os.DirEntry{}
	for _, f := range z.z.File {
		name, err := safeZipPath(f.Name)
		if err != nil {
			continue
		}
		if dir != "" && !strings.HasPrefix(name+"/", dir) {
			continue
		}
		rel := strings.TrimPrefix(name, dir)
		if rel == "" {
			continue
		}
		if strings.HasSuffix(f.Name, "/") {
			base := path.Base(strings.TrimSuffix(rel, "/"))
			if base != "" {
				entries[base] = ZipDirEntry{file: f}
			}
			continue
		}

		// Directory intermedie implicite.
		parts := strings.Split(rel, "/")
		if len(parts) > 1 {
			dirName := parts[0]
			if _, ok := entries[dirName]; !ok {
				entries[dirName] = ZipDirEntry{
					file: &zip.File{
						FileHeader: zip.FileHeader{
							Name: dir + dirName + "/",
						},
					},
				}
			}
		}

		// File immediato.
		if len(parts) == 1 {
			entries[rel] = ZipDirEntry{file: f}
		}
	}

	out := make([]os.DirEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, e)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Name() < out[j].Name()
	})

	return out, nil
}
