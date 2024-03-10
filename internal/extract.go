package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
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

	metaFilename := path.Join(cwd, "meta.json")
	if _, err = os.Stat(metaFilename); err != nil {
		var data []byte
		data, err = json.MarshalIndent(probe, "", " ")
		if err != nil {
			return
		}

		err = os.WriteFile(metaFilename, data, FILE_PERM)
		if err != nil {
			return
		}
	}

	var processedStreams []ProcessedStream
	if processedStreams, err = FfmpegExtractStreams(cwd, filepath, probe.Streams); err != nil {
		return
	}

	for _, stream := range getStreamsByType(probe.Streams, SUBTITLE_CODEC) {
		var filename string
		if filename, err = FfmpegExtractSubtitleStream(cwd, filepath, stream); err != nil {
			return
		}
		processedStreams = append(processedStreams, ProcessedStream{
			filename: filename,
			stream:   stream,
		})
	}

	err = BuildMain(cwd, processedStreams)

	return
}

func BuildMain(cwd string, processedStreams []ProcessedStream) (err error) {
	filename := path.Join(cwd, "main.m3u8")
	if _, err = os.Stat(filename); err == nil {
		log.Printf("Main file exists, skip\n")
		return
	}

	var lines = []string{
		"#EXTM3U",
	}

	for _, f := range processedStreams {
		switch f.stream.CodecType {
		case VIDEO_CODEC:
			filename := path.Base(f.filename)
			lines = append(lines, "#EXT-X-STREAM-INF:PROGRAM-ID=1", filename)
		case AUDIO_CODEC:
			filename := path.Base(f.filename)
			name := getStreamName(f.stream)
			lines = append(lines, "#EXT-X-MEDIA:TYPE=AUDIO,NAME=\""+name+"\",URI=\""+url.QueryEscape(filename)+"\"")
		case SUBTITLE_CODEC:
			filename := path.Base(f.filename)
			name := getStreamName(f.stream)
			lines = append(lines, "#EXT-X-MEDIA:TYPE=SUBTITLES,NAME=\""+name+"\",URI=\""+url.QueryEscape(filename)+"\"")
		}
	}

	data := strings.Join(lines, "\n")
	err = os.WriteFile(filename, []byte(data), FILE_PERM)

	return
}

func getStreamName(stream *ProbeStream) string {
	var parts []string
	if language, ok := stream.Tags["language"]; ok {
		parts = append(parts, language)
	}
	if title, ok := stream.Tags["title"]; ok {
		parts = append(parts, title)
	}
	if len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%d", stream.Index))
	}
	return strings.Join(parts, " - ")
}
