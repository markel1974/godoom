package lumps

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
)

// NodePk3 represents a hierarchical node structure for managing PK3 archive files.
// Each node can have a path, child nodes, and an optional zip.File entry.
type NodePk3 struct {
	path     []string
	children map[string]*NodePk3
	entry    *zip.File
}

// NewPk3Node initializes and returns a new NodePk3 with the provided path and an empty children map.
func NewPk3Node(path []string) *NodePk3 {
	return &NodePk3{
		path:     path,
		children: make(map[string]*NodePk3),
	}
}

// GetChildren returns a sorted list of child node names from the current NodePk3 instance.
func (np *NodePk3) GetChildren() []string {
	var entries []string
	for name := range np.children {
		part := name
		entries = append(entries, part)
	}
	return entries
}

// GetReader opens and reads the associated zip entry, returning an io.ReadSeeker for the file's content or an error.
func (np *NodePk3) GetReader() (io.ReadSeeker, error) {
	if np.entry == nil {
		p := strings.Join(np.path, PakSeparator)
		return nil, fmt.Errorf("file %s not found in PK3 or is a directory", p)
	}
	rc, err := np.entry.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

// AddNode inserts a file or directory path into the node tree, creating intermediate nodes if necessary.
func (np *NodePk3) AddNode(name string, entry *zip.File) {
	fullPath := np.pathCleaner(name)
	parts := strings.Split(fullPath, PakSeparator)

	curr := np
	for i, part := range parts {
		if _, ok := curr.children[part]; !ok {
			curr.children[part] = NewPk3Node(append(curr.path, part))
		}
		curr = curr.children[part]
		if i == len(parts)-1 {
			curr.entry = entry
		}
	}
}

// GetNode navigates to a specific node in the tree based on the given path segments and returns it, or nil if not found.
func (np *NodePk3) GetNode(p string) *NodePk3 {
	fullPath := np.pathCleaner(p)
	parts := strings.Split(fullPath, PakSeparator)
	curr := np
	if len(parts) == 0 {
		return curr
	}
	for _, part := range parts {
		if part == "" {
			continue
		}
		child, ok := curr.children[part]
		if !ok {
			return nil
		}
		curr = child
	}
	return curr
}

// pathCleaner normalizes a given file or directory path by trimming whitespace and converting it to lowercase.
func (np *NodePk3) pathCleaner(in string) string {
	p := strings.ReplaceAll(in, "\\", PakSeparator)
	return strings.TrimSpace(strings.ToLower(p))
}

// Pk3 represents a structure for handling and navigating through PK3 archive files.
type Pk3 struct {
	root   *NodePk3
	reader *zip.ReadCloser
}

// NewPk3 creates and returns a new instance of the Pk3 structure for handling PK3 archives.
func NewPk3() *Pk3 {
	return &Pk3{}
}

// Setup initializes the Pk3 structure by reading a PK3 file and building an in-memory representation of its contents.
func (pk *Pk3) Setup(path string) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	pk.reader = r

	pk.root = NewPk3Node(nil)
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		pk.root.AddNode(f.Name, f)
	}
	return nil
}

// Open retrieves a file by its full path within the PK3 archive and returns an io.ReadSeeker for reading its content.
func (pk *Pk3) Open(fullPath string) (io.ReadSeeker, error) {
	node := pk.root.GetNode(fullPath)
	if node == nil {
		return nil, fmt.Errorf("file %s not found in PK3", fullPath)
	}
	return node.GetReader()
}

// ReadDir retrieves the list of child nodes (files/directories) at the specified directory path within the PK3 file.
// If the path does not exist or is a file, an error is returned.
func (pk *Pk3) ReadDir(fullPath string) ([]string, error) {
	node := pk.root.GetNode(fullPath)
	if node == nil {
		return nil, fmt.Errorf("directory not found: %s", fullPath)
	}
	if node.entry != nil {
		return nil, fmt.Errorf("path is a file, not a directory: %s", fullPath)
	}
	return node.GetChildren(), nil
}

// ReadDirFilter lists directory entries matching a wildcard pattern within a PK3 archive at the given path.
func (pk *Pk3) ReadDirFilter(fullPath string, wildcard string) ([]string, error) {
	r, err := regexp.Compile(wildcard)
	if err != nil {
		return nil, err
	}
	entries, err := pk.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, entry := range entries {
		if r.MatchString(entry) {
			out = append(out, entry)
		}
	}
	sort.Strings(out)
	return out, nil
}
