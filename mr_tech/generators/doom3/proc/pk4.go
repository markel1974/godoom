package proc

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/markel1974/godoom/mr_tech/generators/common"
)

const Pak4Separator = "/"

// NodePk4 represents a node in a hierarchical structure for handling zip file contents.
// It includes the path to the node, its children, and an associated zip.File entry if applicable.
type NodePk4 struct {
	path     []string
	children map[string]*NodePk4
	entry    *zip.File
}

// NewPk3Node creates and returns a new NodePk4 with the given path and initializes its children as an empty map.
func NewPk3Node(path []string) *NodePk4 {
	return &NodePk4{
		path:     path,
		children: make(map[string]*NodePk4),
	}
}

// GetChildren retrieves the names of all immediate child nodes stored in the `children` map of the current node.
func (np *NodePk4) GetChildren() []string {
	var entries []string
	for name := range np.children {
		part := name
		entries = append(entries, part)
	}
	return entries
}

// GetReader returns an io.ReadSeeker for the file referenced by the NodePk4 or an error if the node is not a valid file.
func (np *NodePk4) GetReader() (io.ReadSeeker, error) {
	if np.entry == nil {
		p := strings.Join(np.path, Pak4Separator)
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

// AddNode inserts a file or directory into the node hierarchy based on the provided path and associates it with the given zip entry.
// The method creates intermediary nodes as necessary to reflect the full path structure.
func (np *NodePk4) AddNode(name string, entry *zip.File) {
	fullPath := np.pathCleaner(name)
	parts := strings.Split(fullPath, Pak4Separator)

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

// GetNode retrieves a child node from the current node by parsing the provided path string. Returns nil if not found.
func (np *NodePk4) GetNode(p string) *NodePk4 {
	fullPath := np.pathCleaner(p)
	parts := strings.Split(fullPath, Pak4Separator)
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

// pathCleaner normalizes a given path by replacing backslashes with the path separator and converting it to lowercase trim.
func (np *NodePk4) pathCleaner(in string) string {
	p := strings.ReplaceAll(in, "\\", Pak4Separator)
	return strings.TrimSpace(strings.ToLower(p))
}

// Pk4 represents a package loader for managing and extracting files from zip-based archives (e.g., PK4 files).
type Pk4 struct {
	res     common.IResources
	root    *NodePk4
	reader  *zip.Reader
	readers []*zip.Reader
}

// NewPk4 creates a new instance of Pk4 using the provided IResources interface implementation.
func NewPk4(res common.IResources) *Pk4 {
	return &Pk4{
		res: res,
	}
}

// Setup initializes the Pk4 instance by loading and parsing the resources from the provided zip file path.
// It constructs an internal directory structure to manage the file hierarchy within the zip.
func (pk *Pk4) Setup(path string) error {
	pk.root = NewPk3Node(nil)

	// Collect any sibling pk4 files in the directory so multi-part assets (pak000..pak004) are mounted together
	dir := filepath.Dir(path)
	var pk4Files []string
	if entries, err := pk.res.ReadDir(dir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".pk4") {
				pk4Files = append(pk4Files, filepath.Join(dir, entry.Name()))
			}
		}
		sort.Strings(pk4Files)
	}

	if len(pk4Files) == 0 {
		pk4Files = []string{path}
	}

	for _, p := range pk4Files {
		k, err := pk.res.Open(p)
		if err != nil {
			continue
		}
		size, err := k.Seek(0, io.SeekEnd)
		if err != nil {
			continue
		}
		if _, err = k.Seek(0, io.SeekStart); err != nil {
			continue
		}
		r, err := zip.NewReader(k, size)
		if err != nil {
			continue
		}
		pk.readers = append(pk.readers, r)
		pk.reader = r
		for _, f := range r.File {
			if f.FileInfo().IsDir() {
				continue
			}
			pk.root.AddNode(f.Name, f)
		}
	}

	if len(pk.readers) == 0 {
		return fmt.Errorf("failed to open pk4 archive %s", path)
	}

	return nil
}

// Open attempts to locate and open a file inside the PK4 archive given its full path. Returns a reader or an error.
func (pk *Pk4) Open(fullPath string) (io.ReadSeeker, error) {
	node := pk.root.GetNode(fullPath)
	if node == nil {
		return nil, fmt.Errorf("file %s not found in PK3", fullPath)
	}
	return node.GetReader()
}

// ReadDir retrieves the names of all entries within a directory at the specified fullPath. Returns an error if the path is invalid or not a directory.
func (pk *Pk4) ReadDir(fullPath string) ([]string, error) {
	node := pk.root.GetNode(fullPath)
	if node == nil {
		return nil, fmt.Errorf("directory not found: %s", fullPath)
	}
	if node.entry != nil {
		return nil, fmt.Errorf("path is a file, not a directory: %s", fullPath)
	}
	return node.GetChildren(), nil
}

// ReadDirFilter filters directory entries at the given path based on a wildcard pattern and returns a sorted list of matches.
// It searches the specified directory and any subdirectories recursively.
func (pk *Pk4) ReadDirFilter(fullPath string, wildcard string) ([]string, error) {
	r, err := regexp.Compile(wildcard)
	if err != nil {
		return nil, err
	}
	node := pk.root.GetNode(fullPath)
	if node == nil {
		return nil, fmt.Errorf("directory not found: %s", fullPath)
	}

	var out []string
	var walk func(n *NodePk4, relPrefix string)
	walk = func(n *NodePk4, relPrefix string) {
		for name, child := range n.children {
			rel := name
			if relPrefix != "" {
				rel = relPrefix + Pak4Separator + name
			}
			if child.entry != nil {
				if r.MatchString(rel) || r.MatchString(name) {
					out = append(out, rel)
				}
			} else {
				walk(child, rel)
			}
		}
	}
	walk(node, "")
	sort.Strings(out)
	return out, nil
}
