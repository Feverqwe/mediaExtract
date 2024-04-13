package main

import (
	"flag"
	"fmt"
	"log"
	"mediaExtract/internal"
	"os"
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

const COMMAND_FILE = "file"
const COMMAND_DIR = "dir"

var COMMAND = []string{COMMAND_FILE, COMMAND_DIR}

func main() {
	var err error

	var meta bool
	var rawFiles arrayFlags
	var aLangs arrayFlags
	var sLangs arrayFlags
	var aMasks arrayFlags
	var sMasks arrayFlags
	var hlsSplitByTime bool
	var hlsTime int
	var hlsSegmentType string
	var hlsMasterPlaylistName string

	offset := 1
	command := COMMAND_FILE
	if len(os.Args) > 1 && internal.ArrayContain(COMMAND, os.Args[1]) {
		offset += 1
		command = os.Args[1]
	}

	f := flag.NewFlagSet(fmt.Sprintf("%s %s", os.Args[0], command), flag.ExitOnError)
	switch command {
	case COMMAND_FILE:
		f.BoolVar(&meta, "meta", false, "Show metadata")
		f.Var(&rawFiles, "f", "Media file")
		f.Var(&aLangs, "al", "Add audio language filter")
		f.Var(&sLangs, "sl", "Add subtilte language filter")
		f.Var(&aMasks, "aMask", "Add audio title mask filter")
		f.Var(&sMasks, "sMask", "Add subtilte title mask filter")
		f.IntVar(&hlsTime, "hlsTime", 10, "Set hls segment time")
		f.BoolVar(&hlsSplitByTime, "hlsSplitByTime", false, "Add hls split by time flag")
		f.StringVar(&hlsSegmentType, "hlsSegmentType", "", "Force set hls segment type: mpegts or fmp4")
		f.StringVar(&hlsMasterPlaylistName, "hlsMasterPlaylistName", "", "Create HLS master playlist with the given name")
	}
	f.Parse(os.Args[offset:])

	if len(rawFiles) == 0 {
		log.Printf("Please provide \"%s\" argument\n", "-f")
		return
	}

	var files []string
	for _, filename := range rawFiles {
		var fullFilename string
		if fullFilename, err = filepath.Abs(filename); err != nil {
			log.Fatal(err)
			return
		}
		files = append(files, fullFilename)
	}

	options := internal.NewOptions(
		aLangs,
		sLangs,
		aMasks,
		sMasks,
		hlsSplitByTime,
		hlsTime,
		hlsSegmentType,
		hlsMasterPlaylistName,
		meta,
	)

	err = internal.Extract(files, options)
	if err != nil {
		log.Fatal(err)
		return
	}
}
