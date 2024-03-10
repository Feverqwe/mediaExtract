package main

import (
	"flag"
	"log"
	"mediaExtract/internal"
	"path/filepath"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "<list>"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	var err error

	var filenameRel string
	var aLangs arrayFlags
	var sLangs arrayFlags

	flag.StringVar(&filenameRel, "f", "", "Media file")
	flag.Var(&aLangs, "al", "Add audio language filter")
	flag.Var(&sLangs, "sl", "Add subtilte language filter")
	flag.Parse()

	filename, err := filepath.Abs(filenameRel)
	if err != nil {
		panic(err)
	}

	err = internal.Extract(filename, aLangs, sLangs)
	if err != nil {
		log.Panic(err)
	}
}
