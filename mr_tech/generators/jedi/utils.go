package jedi

import "unicode"

func CleanKey(in string) string {
	var out []rune
	for _, r := range in {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			out = append(out, unicode.ToUpper(r))
		}
	}
	return string(out)
}
