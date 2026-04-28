package jedi

import (
	"compress/zlib"
	"io"
	"os"
)

func Decompress(name string) {
	in, err := os.Open(name)
	if err != nil {
		panic(err)
	}
	defer in.Close()

	// Salta i 4 byte della signature ZLB e i 4 byte della dimensione
	in.Seek(8, io.SeekStart)

	zr, err := zlib.NewReader(in)
	if err != nil {
		panic(err)
	}
	defer zr.Close()

	out, err := os.Create("jedi.dec")
	if err != nil {
		panic(err)
	}
	defer out.Close()

	io.Copy(out, zr)
}
