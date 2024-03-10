package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
)

type TargetFormat struct {
	codecNames  []string
	codec       string
	codecParams []string
	format      string
	ext         string
}

var CODEC_TARGET_FORMAT = []TargetFormat{
	{
		codecNames: []string{"h264", "hevc"},
		codec:      "copy",
	}, {
		codecNames:  []string{"ac3", "eac3"},
		codec:       "libfdk_aac",
		codecParams: []string{"-vbr", "5"},
	}, {
		codecNames: []string{"subrip"},
		codec:      "webvtt",
		format:     "webvtt",
		ext:        "vtt",
	},
}

type ProbeStream struct {
	Index         int               `json:"index"`
	OrigCodecName string            `json:"orig_codec_name"`
	CodecName     string            `json:"codec_name"`
	CodecType     string            `json:"codec_type"`
	Tags          map[string]string `json:"tags"`
	ChannelLayout string            `json:"channel_layout"`
}

type ProbeFormat struct {
	FormatName string            `json:"format_name"`
	Tags       map[string]string `json:"tags"`
}

type ProbeResult struct {
	Streams []ProbeStream `json:"streams"`
	Format  ProbeFormat   `json:"format"`
}

func ProbeFile(filepath string) (result *ProbeResult, err error) {
	log.Printf("Probe file %s\n", filepath)

	process := exec.Command("ffprobe", "-loglevel", "warning", "-hide_banner", "-i", filepath, "-print_format", "json", "-show_format", "-show_streams")

	process.Env = os.Environ()
	process.Stderr = os.Stderr

	var out strings.Builder
	process.Stdout = &out

	if err = process.Run(); err != nil {
		return
	}

	data := out.String()

	result = &ProbeResult{}
	if err = json.Unmarshal([]byte(data), &result); err != nil {
		return
	}

	return
}

type FloatStream struct {
	name            string
	index           int
	codecTypePrefix string
	codecTypeIdx    int
	stream          *ProbeStream
	codecArgs       []string
	format          string
}

type ProcessedStream struct {
	filename string
	stream   *ProbeStream
}

func FfmpegExtractStreams(cwd, filepath string, probeStreams []ProbeStream, aLangs []string, sLangs []string) (processedStreams []ProcessedStream, err error) {
	var streams []FloatStream
	var codecArgs []string

	for idx, stream := range getStreamsByType(probeStreams, VIDEO_CODEC) {
		index := len(streams)
		name := fmt.Sprintf("%d.m3u8", index)
		if codecArgs, err = getCodecArgs(stream.CodecName); err != nil {
			return
		}
		streams = append(streams, FloatStream{
			name:            name,
			index:           index,
			codecTypePrefix: "v",
			codecTypeIdx:    idx,
			codecArgs:       codecArgs,
			stream:          stream,
		})
	}

	var audioStreamIdx = 0
	for _, stream := range getStreamsByType(probeStreams, AUDIO_CODEC) {
		language := stream.Tags["language"]
		if len(aLangs) > 0 && !ArrayContain(aLangs, language) {
			continue
		}

		index := len(streams)
		typeIndex := audioStreamIdx
		audioStreamIdx++
		name := fmt.Sprintf("%d.m3u8", index)
		if codecArgs, err = getCodecArgs(stream.CodecName); err != nil {
			return
		}
		streams = append(streams, FloatStream{
			name:            name,
			index:           index,
			codecTypePrefix: "a",
			codecTypeIdx:    typeIndex,
			codecArgs:       codecArgs,
			stream:          stream,
		})
	}
	if audioStreamIdx == 0 {
		err = fmt.Errorf("audio streams is empty")
		return
	}

	var subtitleStream []FloatStream
	for _, stream := range getStreamsByType(probeStreams, SUBTITLE_CODEC) {
		language := stream.Tags["language"]
		if len(sLangs) > 0 && !ArrayContain(sLangs, language) {
			continue
		}

		index := len(subtitleStream) + len(streams)

		format, ok := getFormat(stream.CodecName)
		if !ok {
			err = fmt.Errorf("unsupported codec: %s", stream.CodecName)
			return
		}

		name := fmt.Sprintf("%d.%s", index, format.ext)
		subtitleStream = append(subtitleStream, FloatStream{
			index:  index,
			name:   name,
			stream: stream,
			format: format.format,
		})
	}
	if len(subtitleStream) == 0 {
		err = fmt.Errorf("subtitles is empty")
		return
	}

	for _, stream := range append(streams, subtitleStream...) {
		processedStreams = append(processedStreams, ProcessedStream{
			filename: path.Join(cwd, stream.name),
			stream:   stream.stream,
		})
	}

	if _, err = os.Stat(path.Join(cwd, STREAM_POINT)); err == nil {
		return
	}

	const INPUT_INDEX = 0
	args := []string{"-hide_banner", "-y", "-i", filepath}

	for _, stream := range streams {
		mapVal := fmt.Sprintf("%d:%d", INPUT_INDEX, stream.stream.Index)
		args = append(args, "-map", mapVal)
	}

	for _, stream := range streams {
		bitrate := "1"
		if bps, ok := stream.stream.Tags["BPS"]; ok {
			bitrate = bps
		}
		key := fmt.Sprintf("-b:%d", stream.index)
		args = append(args, key, bitrate)
	}

	for _, stream := range streams {
		codecKey := fmt.Sprintf("-codec:%d", stream.index)
		args = append(append(args, codecKey), stream.codecArgs...)
	}

	var varStreamMapItems []string
	for _, stream := range streams {
		val := fmt.Sprintf("%s:%d,agroup:main", stream.codecTypePrefix, stream.codecTypeIdx)
		varStreamMapItems = append(varStreamMapItems, val)
	}
	varStreamMap := strings.Join(varStreamMapItems, " ")

	args = append(args,
		"-f", "hls",
		"-var_stream_map", varStreamMap,
		"-hls_time", "10",
		"-hls_segment_filename", "%v.ts",
		"-hls_segment_type", "fmp4",
		"-hls_flags", "append_list+single_file",
		"-hls_playlist_type", "event",
		"%v.m3u8",
	)

	for _, stream := range subtitleStream {
		index := stream.index
		if codecArgs, err = getCodecArgs(stream.stream.CodecName); err != nil {
			return
		}

		name := stream.name
		mapVal := fmt.Sprintf("%d:%d", INPUT_INDEX, stream.stream.Index)
		codecKey := fmt.Sprintf("-codec:%d", index)
		args = append(args, "-map", mapVal)
		args = append(args, codecKey)
		args = append(args, codecArgs...)
		args = append(args, "-f", stream.format, name)
	}

	log.Printf("Run ffmpeg with args: %v\n", args)

	process := exec.Command("ffmpeg", args...)
	process.Dir = cwd

	process.Env = os.Environ()
	process.Stdout = os.Stdout
	process.Stderr = os.Stderr

	if err = process.Run(); err != nil {
		return
	}

	for _, stream := range subtitleStream {
		plName := fmt.Sprintf("%d.m3u8", stream.index)
		plFilename := path.Join(cwd, plName)

		data := strings.Join([]string{
			"#EXTM3U",
			"#EXT-X-TARGETDURATION:0",
			"#EXT-X-PLAYLIST-TYPE:VOD",
			stream.name,
			"#EXT-X-ENDLIST",
		}, "\n")

		if err = os.WriteFile(plFilename, []byte(data), FILE_PERM); err != nil {
			return
		}
	}

	err = os.WriteFile(path.Join(cwd, STREAM_POINT), []byte("ok"), FILE_PERM)

	return
}

func getCodecArgs(codecName string) (codecArgs []string, err error) {
	format, ok := getFormat(codecName)
	if !ok {
		err = fmt.Errorf("unsupported codec: %s", codecName)
		return
	}

	codecArgs = append(codecArgs, format.codec)
	if len(format.codecParams) > 0 {
		codecArgs = append(codecArgs, format.codecParams...)
	}
	return
}

func getFormat(codecName string) (format TargetFormat, ok bool) {
	for _, f := range CODEC_TARGET_FORMAT {
		for _, c := range f.codecNames {
			if c == codecName {
				format = f
				ok = true
				return
			}
		}
	}
	return
}

func getStreamsByType(streams []ProbeStream, codecType string) (results []*ProbeStream) {
	for i, s := range streams {
		if s.CodecType == codecType {
			results = append(results, &streams[i])
		}
	}
	return
}
