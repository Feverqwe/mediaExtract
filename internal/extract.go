package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
)

func Extract(files []string, options *Options) (err error) {
	if options.target == "" {
		err = errors.New("target_is_empty")
		return
	}

	cwd := options.target
	if _, err = os.Stat(path.Join(cwd, MAIN_PLAYLIST_NAME)); err == nil {
		log.Printf("Main playlist exists \"%s\", skip\n", cwd)
		return
	}

	var probeResults []*ProbeResult
	var streams []ProbeStream
	for idx, filename := range files {
		var probe *ProbeResult
		if probe, err = ProbeFile(idx, filename); err != nil {
			return
		}
		streams = append(streams, probe.Streams...)
		probeResults = append(probeResults, probe)
	}

	var data []byte
	data, err = json.MarshalIndent(probeResults, "", " ")
	if err != nil {
		return
	}

	if options.meta {
		fmt.Println(string(data))
		return
	}

	os.MkdirAll(cwd, DIR_PERM)

	metaFilename := path.Join(cwd, "meta.json")
	err = os.WriteFile(metaFilename, data, FILE_PERM)
	if err != nil {
		return
	}

	err = FfmpegExtractStreams(cwd, files, streams, options)

	return
}
