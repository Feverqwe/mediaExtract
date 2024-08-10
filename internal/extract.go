package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
)

func Extract(files []string, options *Options) (err error) {
	if options.target == "" {
		firstFilename := files[0]
		options.target = GetTargetName(firstFilename)
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
	os.MkdirAll(cwd, DIR_PERM)

	var data []byte
	data, err = json.MarshalIndent(probeResults, "", " ")
	if err != nil {
		return
	}

	if options.meta {
		fmt.Println(string(data))
		return
	}

	metaFilename := path.Join(cwd, "meta.json")
	err = os.WriteFile(metaFilename, data, FILE_PERM)
	if err != nil {
		return
	}

	err = FfmpegExtractStreams(cwd, files, streams, options)

	return
}
