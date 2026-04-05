package lumps

import (
	"errors"
	"io"
	"os"
	"strings"
)

// ToString converts an array of 8 bytes into a string, stopping at the first null byte or the end of the array.
func ToString(s [8]byte) string {
	var i int
	for i = 0; i < len(s); i++ {
		if s[i] == 0 {
			break
		}
	}
	return string(s[:i])
}

// Seek moves the file pointer of the provided file to the specified offset relative to the start of the file.
// Returns an error if the seek operation fails or if the resulting position does not match the requested offset.
func Seek(f *os.File, offset int64) error {
	//off, err := f.Seek(offset, os.SEEK_SET)
	off, err := f.Seek(offset, io.SeekStart)
	if err != nil {
		return err
	}
	if off != offset {
		return errors.New("seek failed")
	}
	return nil
}

// FixName normalizes the input string by converting it to uppercase and trimming whitespace and control characters.
func FixName(in string) string {
	return strings.Trim(strings.ToUpper(in), "\n\r\t ")
}
