package main

import (
	"flag"
	"mediaExtract/internal"
	"path/filepath"
)

func main() {
	var err error

	var filenameRel string
	flag.StringVar(&filenameRel, "f", "", "Media file")

	filename, err := filepath.Abs(filenameRel)
	if err != nil {
		panic(err)
	}

	err = internal.Extract(filename)
	if err != nil {
		panic(err)
	}
}
