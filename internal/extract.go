package internal

import (
	"encoding/json"
	"log"
	"net/url"
	"os"
	"path"
	"strings"
)

type ProcessedFile struct {
	filename string
	stream   *ProbeStream
}

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

	var files []ProcessedFile

	for _, stream := range probe.Streams {
		var filename string
		s := stream
		filename, err = FfmpegExtractStream(cwd, filepath, &s)
		if err != nil {
			return
		}
		files = append(files, ProcessedFile{
			filename: filename,
			stream:   &s,
		})
	}

	if len(files) > 0 {
		err = BuildMain(cwd, files)
	}

	return
}

func BuildMain(cwd string, files []ProcessedFile) (err error) {
	filename := path.Join(cwd, "main.m3u8")
	if _, err = os.Stat(filename); err == nil {
		log.Printf("Main file exists, skip\n")
		return
	}

	var lines = []string{
		"#EXTM3U",
	}

	for _, f := range files {
		switch f.stream.CodecType {
		case VIDEO_CODEC:
			filename := path.Base(f.filename)
			lines = append(lines, "#EXT-X-STREAM-INF:PROGRAM-ID=1", filename)
		case AUDIO_CODEC:
			filename := path.Base(f.filename)
			name := getStreamName(f.stream, filename)
			lines = append(lines, "#EXT-X-MEDIA:TYPE=AUDIO,NAME=\""+name+"\",URI=\""+url.QueryEscape(filename)+"\"")
		case SUBTITLE_CODEC:
			filename := path.Base(f.filename)
			name := getStreamName(f.stream, filename)
			lines = append(lines, "#EXT-X-MEDIA:TYPE=SUBTITLES,NAME=\""+name+"\",URI=\""+url.QueryEscape(filename)+"\"")
		}
	}

	data := strings.Join(lines, "\n")
	err = os.WriteFile(filename, []byte(data), FILE_PERM)

	return
}

func getStreamName(stream *ProbeStream, filename string) string {
	var parts []string
	if language, ok := stream.Tags["language"]; ok {
		parts = append(parts, language)
	}
	if title, ok := stream.Tags["title"]; ok {
		parts = append(parts, title)
	}
	if len(parts) == 0 {
		parts = append(parts, filename)
	}
	return strings.Join(parts, " - ")
}
