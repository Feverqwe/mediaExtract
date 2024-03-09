package internal

import (
	"encoding/json"
	"log"
	"os"
	"path"
	"strings"
)

func Extract(filepath string) (err error) {
	filename := path.Base(filepath)
	name := strings.TrimSuffix(filename, path.Ext(filename))
	placeName := name + ".media"

	probe, err := ProbeFile(filepath)
	if err != nil {
		return
	}

	cwd := path.Join(path.Dir(filepath), placeName)
	os.MkdirAll(cwd, DIR_PERM)

	for _, stream := range probe.Streams {
		if err = ExtractStream(cwd, filepath, &stream); err != nil {
			return
		}

	}

	return
}

func ExtractStream(cwd string, filepath string, stream *ProbeStream) (err error) {
	var filename string
	if filename, err = FfmpegExtractStream(cwd, filepath, stream); err != nil {
		return
	}

	metaFilename := filename + ".json"

	if _, err = os.Stat(metaFilename); err == nil {
		log.Printf("File meta exists, skip\n")
		return
	}

	data, err := json.Marshal(stream)
	if err != nil {
		return
	}

	err = os.WriteFile(metaFilename, data, FILE_PERM)

	return
}
