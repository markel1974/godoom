package jedi

import (
	"fmt"
	"strconv"
	"unicode"
)

func CleanKey(in string) string {
	var out []rune
	for _, r := range in {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			out = append(out, unicode.ToUpper(r))
		}
	}
	return string(out)
}

func GetTokenIntAt(tokens []string, index int) (int, error) {
	if index < 0 || index >= len(tokens) {
		return 0, fmt.Errorf("index out of range")
	}
	count, err := strconv.Atoi(tokens[index])
	if err != nil {
		return 0, err
	}
	return count, nil
}

func GetTokenFloatAt(tokens []string, index int) (float64, error) {
	if index < 0 || index >= len(tokens) {
		return 0, fmt.Errorf("index out of range")
	}
	count, err := strconv.ParseFloat(tokens[index], 64)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func GetTokenStringAt(tokens []string, index int) (string, error) {
	if index < 0 || index >= len(tokens) {
		return "", fmt.Errorf("index out of range")
	}
	return tokens[index], nil
}
