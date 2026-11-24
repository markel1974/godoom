package vertices_element_3d

import "strings"

// CleanString returns s with leading and trailing spaces removed and all letters converted to lowercase.
func CleanString(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}
