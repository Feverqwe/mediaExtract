package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
)

func Extract(files []string, options *Options) (err error) {
	if options.target == "" {
		err = errors.New("target_is_empty")
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

	var cwd = options.target

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
