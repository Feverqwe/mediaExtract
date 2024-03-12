package internal

import (
	"encoding/json"
	"os"
	"path"
	"strings"
)

func Extract(files []string, options Options) (err error) {
	firstFilename := files[0]
	filename := path.Base(firstFilename)
	name := strings.TrimSuffix(filename, path.Ext(filename))
	placeName := name + ".media"

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

	cwd := path.Join(path.Dir(firstFilename), placeName)
	os.MkdirAll(cwd, DIR_PERM)

	metaFilename := path.Join(cwd, "meta.json")
	if _, err = os.Stat(metaFilename); err != nil {
		var data []byte
		data, err = json.MarshalIndent(probeResults, "", " ")
		if err != nil {
			return
		}

		err = os.WriteFile(metaFilename, data, FILE_PERM)
		if err != nil {
			return
		}
	}

	err = FfmpegExtractStreams(cwd, files, streams, options)

	return
}
