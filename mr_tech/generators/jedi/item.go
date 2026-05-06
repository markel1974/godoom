package jedi

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// ItemProperty represents a property of an item with a specific type and value.
type ItemProperty struct {
	Type     string
	Value    float64
	ValueStr string
}

// Item represents an entity with a unique name, a specific function, a model, and a collection of properties.
type Item struct {
	Name       string
	Version    string
	Function   string
	Anim       string
	Sound      string
	Model      string
	Data       int
	Properties map[string]ItemProperty
}

// NewItem creates and returns a new instance of Item with an initialized Properties map.
func NewItem() *Item {
	return &Item{
		Properties: make(map[string]ItemProperty),
	}
}

// Parse populates the Item instance by parsing key-value pairs and structured data from the provided tokens slice.
func (it *Item) Parse(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		tokens := strings.Fields(line)
		for i := 0; i < len(tokens); i++ {
			key := strings.ToUpper(tokens[i])
			switch key {
			case "ITEM":
				i++
				it.Version, _ = GetTokenStringAt(tokens, i)
			case "NAME":
				i++
				it.Name, _ = GetTokenStringAt(tokens, i)
			case "FUNC":
				i++
				it.Function, _ = GetTokenStringAt(tokens, i)
			case "ANIM":
				i++
				it.Anim, _ = GetTokenStringAt(tokens, i)
			case "MODEL":
				i++
				it.Model, _ = GetTokenStringAt(tokens, i)
			case "SOUND":
				i++
				it.Sound, _ = GetTokenStringAt(tokens, i)
			case "STR", "INT", "FLOAT":
				i++
				k, _ := GetTokenStringAt(tokens, i)
				i++
				parsedStr, _ := GetTokenStringAt(tokens, i)
				it.Properties[k] = ItemProperty{Type: key, Value: 0, ValueStr: parsedStr}
			case "DATA":
				i++
				it.Data, _ = GetTokenIntAt(tokens, i)
			default:
				fmt.Println("Unknown ITM property:", strings.Join(tokens, "||"))
			}
			i = len(tokens)
		}
	}
	return nil
}

// GetInt retrieves the integer value associated with the provided key from the Item's properties or returns the default value.
func (it *Item) GetInt(key string) (int, bool) {
	v, ok := it.GetFloat(key)
	if !ok {
		return 0, false
	}
	return int(v), true
}

// GetFloat retrieves the float64 value associated with the given key from the item's properties or returns the default value.
func (it *Item) GetFloat(key string) (float64, bool) {
	if p, ok := it.Properties[strings.ToUpper(key)]; ok {
		return p.Value, true
	}
	return 0, false
}

// GetString retrieves the string value of a property by its key. Returns defaultVal if the key does not exist or is not a string.
func (it *Item) GetString(key string) (string, bool) {
	if p, ok := it.Properties[strings.ToUpper(key)]; ok {
		return p.ValueStr, true
	}
	return "", false
}
