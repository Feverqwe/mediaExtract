package main

import (
	"flag"
	"fmt"
	"log"
	"mediaExtract/internal"
	"os"
	"path/filepath"
	"strings"
)

const COMMAND_FILE = "file"
const COMMAND_DIR = "dir"

var COMMAND = []string{COMMAND_FILE, COMMAND_DIR}

func main() {
	var err error

	offset := 1
	command := COMMAND_FILE
	if len(os.Args) > 1 && internal.ArrayContain(COMMAND, os.Args[1]) {
		offset += 1
		command = os.Args[1]
	}

	args := os.Args[offset:]

	switch command {
	case COMMAND_FILE:
		err = runFile(args)
	case COMMAND_DIR:
		err = runDir(args)
	}

	if err != nil {
		log.Fatal(err)
		return
	}
}

func runFile(args []string) (err error) {
	var rawFiles internal.ArrayFlags
	var rawTarget string

	command := COMMAND_FILE

	f := flag.NewFlagSet(fmt.Sprintf("%s %s", os.Args[0], command), flag.ExitOnError)

	basicOptions := internal.GetBasicOptions(f)
	f.Var(&rawFiles, "f", "Media file")
	f.StringVar(&rawTarget, "t", "", "Traget folder")

	f.Parse(args)

	if len(rawFiles) == 0 {
		log.Printf("Please provide \"%s\" argument\n", "-f")
		return
	}

	var target string
	if rawTarget != "" {
		if target, err = filepath.Abs(rawTarget); err != nil {
			log.Fatal(err)
			return
		}
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

	options := internal.NewFileOptions(
		basicOptions,
		target,
	)

	err = internal.Extract(files, &options)
	return
}

func runDir(args []string) (err error) {
	command := COMMAND_DIR

	f := flag.NewFlagSet(fmt.Sprintf("%s %s", os.Args[0], command), flag.ExitOnError)

	var directory string
	var patterns internal.ArrayFlags
	var deepLevel int

	basicOptions := internal.GetBasicOptions(f)
	f.StringVar(&directory, "d", "", "Media directory")
	f.Var(&patterns, "p", "File name patterns")
	f.IntVar(&deepLevel, "l", 0, "Subdirectory deep level")

	f.Parse(args)

	if len(directory) == 0 {
		log.Printf("Please provide \"%s\" argument\n", "-d")
		return
	}

	if len(patterns) == 0 {
		log.Printf("Please provide \"%s\" argument\n", "-p")
		return
	}

	dirOffset := 1
	if strings.HasSuffix(directory, string(filepath.Separator)) {
		dirOffset -= 1
	}

	var allFiles []string
	level := ""
	for i := 0; i <= deepLevel; i++ {
		for _, pattern := range patterns {
			var files []string
			files, err = filepath.Glob(filepath.Join(directory, level, pattern))
			if err != nil {
				return
			}
			allFiles = append(allFiles, files...)
		}
		level += "*" + string(filepath.Separator)
	}

	for _, filename := range allFiles {
		relFilename := filename[len(directory)+dirOffset:]
		log.Printf("Processing file \"%s\"\n", relFilename)
		target := ""
		options := internal.NewFileOptions(
			basicOptions,
			target,
		)
		err = internal.Extract([]string{filename}, &options)
		if err != nil {
			return
		}
	}

	return
}
