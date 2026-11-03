package proc

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Lexer is a tokenizer for Doom 3 ASCII definition files (.proc, .map, .mtr).
type Lexer struct {
	data []byte
	pos  int
	len  int
	peek string
	hasP bool
}

// NewLexer creates a new lexer from a byte slice.
func NewLexer(data []byte) *Lexer {
	return &Lexer{
		data: data,
		pos:  0,
		len:  len(data),
	}
}

// NextToken returns the next lexical token, skipping whitespace and comments.
func (l *Lexer) NextToken() (string, error) {
	if l.hasP {
		l.hasP = false
		return l.peek, nil
	}

	for {
		l.skipWhitespace()
		if l.pos >= l.len {
			return "", io.EOF
		}

		// Line comment //
		if l.pos+1 < l.len && l.data[l.pos] == '/' && l.data[l.pos+1] == '/' {
			l.pos += 2
			for l.pos < l.len && l.data[l.pos] != '\n' {
				l.pos++
			}
			continue
		}

		// Block comment /* ... */
		if l.pos+1 < l.len && l.data[l.pos] == '/' && l.data[l.pos+1] == '*' {
			l.pos += 2
			for l.pos+1 < l.len && !(l.data[l.pos] == '*' && l.data[l.pos+1] == '/') {
				l.pos++
			}
			if l.pos+1 < l.len {
				l.pos += 2
			} else {
				l.pos = l.len
			}
			continue
		}

		break
	}

	if l.pos >= l.len {
		return "", io.EOF
	}

	b := l.data[l.pos]

	// Single-character delimiters
	if b == '{' || b == '}' || b == '(' || b == ')' {
		l.pos++
		return string([]byte{b}), nil
	}

	// Quoted string
	if b == '"' {
		l.pos++
		start := l.pos
		for l.pos < l.len && l.data[l.pos] != '"' {
			if l.data[l.pos] == '\\' && l.pos+1 < l.len {
				l.pos++
			}
			l.pos++
		}
		val := string(l.data[start:l.pos])
		if l.pos < l.len && l.data[l.pos] == '"' {
			l.pos++
		}
		return val, nil
	}

	// General token (identifiers, numbers, paths)
	start := l.pos
	for l.pos < l.len {
		c := l.data[l.pos]
		if c <= ' ' || c == '{' || c == '}' || c == '(' || c == ')' || c == '"' {
			break
		}
		l.pos++
	}
	if start == l.pos && l.pos < l.len {
		l.pos++
		return string([]byte{b}), nil
	}

	return string(l.data[start:l.pos]), nil
}

// PeekToken looks ahead at the next token without advancing the lexer position.
func (l *Lexer) PeekToken() (string, error) {
	if l.hasP {
		return l.peek, nil
	}
	tok, err := l.NextToken()
	if err != nil {
		return "", err
	}
	l.peek = tok
	l.hasP = true
	return tok, nil
}

// Expect consumes the next token and verifies that it matches the expected string.
func (l *Lexer) Expect(expected string) error {
	tok, err := l.NextToken()
	if err != nil {
		return fmt.Errorf("expected '%s', got EOF: %w", expected, err)
	}
	if tok != expected {
		return fmt.Errorf("expected '%s', got '%s'", expected, tok)
	}
	return nil
}

// ReadFloat parses and returns the next token as a float64.
func (l *Lexer) ReadFloat() (float64, error) {
	tok, err := l.NextToken()
	if err != nil {
		return 0, err
	}
	v, err := strconv.ParseFloat(tok, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid float '%s': %w", tok, err)
	}
	return v, nil
}

// ReadInt parses and returns the next token as an integer.
func (l *Lexer) ReadInt() (int, error) {
	tok, err := l.NextToken()
	if err != nil {
		return 0, err
	}
	v, err := strconv.Atoi(tok)
	if err != nil {
		return 0, fmt.Errorf("invalid int '%s': %w", tok, err)
	}
	return v, nil
}

// SkipBlock skips tokens until the matching closing brace '}' for a block starting with '{'.
func (l *Lexer) SkipBlock() error {
	if l.hasP {
		if l.peek == "{" {
			l.hasP = false
		} else if l.peek == "}" {
			l.hasP = false
			return nil
		} else {
			l.hasP = false
		}
	}

	depth := 1
	for l.pos < l.len {
		b := l.data[l.pos]

		// Line comment //
		if b == '/' && l.pos+1 < l.len && l.data[l.pos+1] == '/' {
			l.pos += 2
			for l.pos < l.len && l.data[l.pos] != '\n' {
				l.pos++
			}
			continue
		}

		// Block comment /* ... */
		if b == '/' && l.pos+1 < l.len && l.data[l.pos+1] == '*' {
			l.pos += 2
			for l.pos+1 < l.len && !(l.data[l.pos] == '*' && l.data[l.pos+1] == '/') {
				l.pos++
			}
			if l.pos+1 < l.len {
				l.pos += 2
			} else {
				l.pos = l.len
			}
			continue
		}

		// String literal "..."
		if b == '"' {
			l.pos++
			for l.pos < l.len && l.data[l.pos] != '"' {
				if l.data[l.pos] == '\\' && l.pos+1 < l.len {
					l.pos++
				}
				l.pos++
			}
			if l.pos < l.len {
				l.pos++
			}
			continue
		}

		if b == '{' {
			depth++
		} else if b == '}' {
			depth--
			if depth <= 0 {
				l.pos++
				return nil
			}
		}

		l.pos++
	}

	return io.EOF
}

func (l *Lexer) skipWhitespace() {
	for l.pos < l.len && l.data[l.pos] <= ' ' {
		l.pos++
	}
}

// StripQuotes removes enclosing double quotes from a token if present.
func StripQuotes(s string) string {
	return strings.Trim(s, "\"")
}
