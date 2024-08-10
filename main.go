package main

import (
	"flag"
	"fmt"
	"log"
	"mediaExtract/internal"
	"os"
	"path"
	"path/filepath"
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

	if target == "" {
		firstFilename := files[0]
		target = internal.GetTargetName(firstFilename, "")
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

	var rawDirectory string
	var rawTargetDirectory string
	var patterns internal.ArrayFlags
	var deepLevel int

	basicOptions := internal.GetBasicOptions(f)
	f.StringVar(&rawDirectory, "d", "", "Media directory")
	f.StringVar(&rawTargetDirectory, "td", "", "Target directory")
	f.Var(&patterns, "p", "File name patterns")
	f.IntVar(&deepLevel, "l", 0, "Subdirectory deep level")

	f.Parse(args)

	if len(rawDirectory) == 0 {
		log.Printf("Please provide \"%s\" argument\n", "-d")
		return
	}

	if len(patterns) == 0 {
		log.Printf("Please provide \"%s\" argument\n", "-p")
		return
	}

	var directory string
	if directory, err = filepath.Abs(rawDirectory); err != nil {
		log.Fatal(err)
		return
	}

	targetDirectory := directory
	if rawTargetDirectory != "" {
		if targetDirectory, err = filepath.Abs(rawTargetDirectory); err != nil {
			log.Fatal(err)
			return
		}
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
		relFilename := filename[len(directory)+1:]
		relFileDir := path.Dir(relFilename)
		log.Printf("Processing file \"%s\"\n", relFilename)

		targetDir := path.Join(targetDirectory, relFileDir)
		target := internal.GetTargetName(filename, targetDir)
		if _, err = os.Stat(target); err == nil {
			log.Printf("Target folder \"%s\" exists, skip\n", target)
			continue
		}

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
