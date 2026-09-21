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

type NodePk3 struct {
	path     []string
	children map[string]*NodePk3
	entry    *zip.File
}

func NewPk3Node(path []string) *NodePk3 {
	return &NodePk3{
		path:     path,
		children: make(map[string]*NodePk3),
	}
}

func (np *NodePk3) GetChildren() []string {
	var entries []string
	for name := range np.children {
		entries = append(entries, name)
	}
	return entries
}

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

func (np *NodePk3) AddNode(parts []string, entry *zip.File) {
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

func (np *NodePk3) GetNode(parts []string) *NodePk3 {
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

type Pk3 struct {
	root   *NodePk3
	reader *zip.ReadCloser
}

func NewPk3() *Pk3 {
	return &Pk3{}
}

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
		fullPath := strings.ReplaceAll(f.Name, "\\", PakSeparator)
		parts := strings.Split(fullPath, PakSeparator)
		pk.root.AddNode(parts, f)
	}
	return nil
}

func (pk *Pk3) Open(fullPath string) (io.ReadSeeker, error) {
	parts := strings.Split(fullPath, PakSeparator)
	node := pk.root.GetNode(parts)
	if node == nil {
		return nil, fmt.Errorf("file %s not found in PK3", fullPath)
	}
	return node.GetReader()
}

func (pk *Pk3) ReadDir(fullPath string) ([]string, error) {
	parts := strings.Split(fullPath, PakSeparator)
	node := pk.root.GetNode(parts)
	if node == nil {
		return nil, fmt.Errorf("directory not found: %s", fullPath)
	}
	if node.entry != nil {
		return nil, fmt.Errorf("path is a file, not a directory: %s", fullPath)
	}
	return node.GetChildren(), nil
}

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
