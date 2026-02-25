package lumps

import (
	"errors"
	"io"
	"os"
	"strings"
)

func ToString(s [8]byte) string {
	var i int
	for i = 0; i < len(s); i++ {
		if s[i] == 0 {
			break
		}
	}
	return string(s[:i])
}

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

func FixName(in string) string {
	return strings.Trim(strings.ToUpper(in), "\n\r\t ")
}
