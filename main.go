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

	var filename string
	var aLangs arrayFlags
	var sLangs arrayFlags
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
		f.StringVar(&filename, "f", "", "Media file")
		f.Var(&aLangs, "al", "Add audio language filter")
		f.Var(&sLangs, "sl", "Add subtilte language filter")
		f.IntVar(&hlsTime, "hlsTime", 10, "Set hls segment time")
		f.BoolVar(&hlsSplitByTime, "hlsSplitByTime", false, "Add hls split by time flag")
		f.StringVar(&hlsSegmentType, "hlsSegmentType", "", "Force set hls segment type: mpegts or fmp4")
		f.StringVar(&hlsMasterPlaylistName, "hlsMasterPlaylistName", "", "Create HLS master playlist with the given name")
	}
	f.Parse(os.Args[offset:])

	if filename == "" {
		log.Printf("Please provide \"%s\" argument\n", "-f")
		return
	}

	if filename, err = filepath.Abs(filename); err != nil {
		log.Fatal(err)
		return
	}

	options := internal.NewOptions(
		aLangs,
		sLangs,
		hlsSplitByTime,
		hlsTime,
		hlsSegmentType,
		hlsMasterPlaylistName,
	)

	err = internal.Extract(filename, options)
	if err != nil {
		log.Fatal(err)
		return
	}
}
