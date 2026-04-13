package lumps

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

const PakSeparator = "/"

// EntryPAK represents a file entry within a PAK archive, containing metadata such as file, name, offset, and size.
type EntryPAK struct {
	file   *os.File
	offset int64
	size   int64
}

// NewEntryPak creates a new EntryPAK instance with the specified file, name, offset, and size.
func NewEntryPak(file *os.File, offset int64, size int64) *EntryPAK {
	return &EntryPAK{
		file:   file,
		offset: offset,
		size:   size,
	}
}

// GetReader returns an io.Reader for accessing a specific section of the file represented by the EntryPAK instance.
func (ep *EntryPAK) GetReader() io.ReadSeeker {
	return io.NewSectionReader(ep.file, ep.offset, ep.size)
}

// NodePak represents a node in a hierarchical structure, with support for children and an optional associated entry.
type NodePak struct {
	path     []string
	children map[string]*NodePak
	entry    *EntryPAK
}

// NewPakNode creates a new NodePak with the specified path and initializes its children as an empty map.
func NewPakNode(path []string) *NodePak {
	return &NodePak{
		path:     path,
		children: make(map[string]*NodePak),
	}
}

// Path returns the path of the current NodePak as a slice of strings.
func (np *NodePak) Path() []string {
	return np.path
}

// GetChildren returns the names of all child nodes under the current NodePak.
func (np *NodePak) GetChildren() []string {
	var entries []string
	for name := range np.children {
		entries = append(entries, name)
	}
	return entries
}

// GetReader retrieves an io.Reader for the underlying entry if available or returns an error if the entry is nil.
func (np *NodePak) GetReader() (io.ReadSeeker, error) {
	if np.entry == nil {
		p := strings.Join(np.path, PakSeparator)
		return nil, fmt.Errorf("file %s not found in PAK or is a directory", p)
	}
	return np.entry.GetReader(), nil
}

// AddNode inserts a new node into the hierarchical structure using the provided path and EntryPAK.
func (np *NodePak) AddNode(parts []string, entry *EntryPAK) {
	curr := np
	for i, part := range parts {
		if _, ok := curr.children[part]; !ok {
			curr.children[part] = NewPakNode(append(curr.path, part))
		}
		curr = curr.children[part]
		if i == len(parts)-1 {
			curr.entry = entry
		}
	}
}

// GetNode retrieves a child node from the given path or returns nil if the path does not exist.
func (np *NodePak) GetNode(parts []string) *NodePak {
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

// Pak represents a container for a hierarchical structure of files and directories.
type Pak struct {
	root *NodePak
}

// NewPak initializes and returns a new instance of the Pak structure.
func NewPak() *Pak {
	return &Pak{}
}

// Setup initializes the Pak structure by reading the PAK file at the given path and building its directory hierarchy.
func (pk *Pak) Setup(path string) error {
	type Header struct {
		Magic     [4]byte
		DirOffset int32
		DirSize   int32
	}
	type Entry struct {
		Name   [56]byte
		Offset int32
		Size   int32
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	var header Header
	if err = binary.Read(file, binary.LittleEndian, &header); err != nil {
		return err
	}
	if string(header.Magic[:]) != "PACK" {
		return fmt.Errorf("invalid PAK magic")
	}
	if _, err = file.Seek(int64(header.DirOffset), io.SeekStart); err != nil {
		return err
	}
	numEntries := int(header.DirSize) / 64
	pk.root = NewPakNode(nil)
	for i := 0; i < numEntries; i++ {
		reader := &Entry{}
		if err = binary.Read(file, binary.LittleEndian, reader); err != nil {
			return err
		}
		fullPath := FromNullTerminatingString(reader.Name[:])
		entry := NewEntryPak(file, int64(reader.Offset), int64(reader.Size))
		parts := pk.Parts(fullPath)
		pk.root.AddNode(parts, entry)
	}
	return nil
}

// Open retrieves a file or resource from the PAK archive by its name and returns a reader to access its contents.
func (pk *Pak) Open(fullPath string) (io.ReadSeeker, error) {
	parts := pk.Parts(fullPath)
	node := pk.root.GetNode(parts)
	if node == nil {
		return nil, fmt.Errorf("file %s not found in PAK or is a directory", fullPath)
	}
	return node.GetReader()
}

// ReadDir retrieves the names of all child nodes within the specified directory path in the PAK file.
// Returns an error if the path does not exist or points to a file instead of a directory.
func (pk *Pak) ReadDir(fullPath string) ([]string, error) {
	parts := pk.Parts(fullPath)
	node := pk.root.GetNode(parts)
	if node == nil {
		return nil, fmt.Errorf("directory not found: %s", fullPath)
	}
	if node.entry != nil {
		return nil, fmt.Errorf("path is a file, not a directory: %s", fullPath)
	}
	return node.GetChildren(), nil
}

// Parts splits the given path string into its individual components using the NodePak's separator and returns them as a slice.
func (pk *Pak) Parts(path string) []string {
	parts := strings.Split(path, PakSeparator)
	return parts
}
