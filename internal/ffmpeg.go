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
}

var CODEC_TARGET_FORMAT = []TargetFormat{
	{
		codecNames: []string{"h264", "hevc"},
		codec:      "copy",
	},
	{
		codecNames: []string{"subrip"},
		codec:      "webvtt",
		/* format:     "webvtt",
		ext:        "vtt",
		postProcess: func(cwd, filepath string) (filename string, err error) {
			filename = strings.TrimSuffix(filepath, path.Ext(filepath)) + ".m3u8"
			if _, err = os.Stat(filename); err == nil {
				return
			}

			data := strings.Join([]string{
				"#EXTM3U",
				"#EXT-X-TARGETDURATION:0",
				"#EXT-X-PLAYLIST-TYPE:VOD",
				path.Base(filepath),
				"#EXT-X-ENDLIST",
			}, "\n")
			err = os.WriteFile(filename, []byte(data), FILE_PERM)
			return
		}, */
	}, {
		codecNames:  []string{"ac3", "eac3"},
		codec:       "libfdk_aac",
		codecParams: []string{"-vbr", "5"},
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

	process := exec.Command("ffprobe", "-hide_banner", "-i", filepath, "-print_format", "json", "-show_format", "-show_streams")

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
	index           int
	codecTypePrefix string
	codecTypeIdx    int
	stream          *ProbeStream
	codecArgs       []string
}

func FfmpegExtractStreams(cwd, filepath string, probeStreams []ProbeStream) (err error) {
	var streams []FloatStream
	var codecArgs []string

	for idx, stream := range getStreamsByType(probeStreams, VIDEO_CODEC) {
		if codecArgs, err = getCodecArgs(stream.CodecName); err != nil {
			return
		}
		streams = append(streams, FloatStream{
			index:           len(streams),
			codecTypePrefix: "v",
			codecTypeIdx:    idx,
			codecArgs:       codecArgs,
			stream:          stream,
		})
	}

	for idx, stream := range getStreamsByType(probeStreams, AUDIO_CODEC) {
		if codecArgs, err = getCodecArgs(stream.CodecName); err != nil {
			return
		}
		streams = append(streams, FloatStream{
			index:           len(streams),
			codecTypePrefix: "a",
			codecTypeIdx:    idx,
			codecArgs:       codecArgs,
			stream:          stream,
		})
	}

	/* for idx, stream := range getStreamsByType(probeStreams, SUBTITLE_CODEC) {
		if codecArgs, err = getCodecArgs(stream.CodecName); err != nil {
			return
		}
		streams = append(streams, FloatStream{
			index:           len(streams),
			codecTypePrefix: "s",
			codecTypeIdx:    idx,
			codecArgs:       codecArgs,
			stream:          stream,
		})
	} */

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
		// name := getStreamName(stream.stream)
		val := fmt.Sprintf("%s:%d,agroup:main", stream.codecTypePrefix, stream.codecTypeIdx)
		varStreamMapItems = append(varStreamMapItems, val)
	}
	varStreamMap := strings.Join(varStreamMapItems, " ")

	filename := path.Join(cwd, "main.m3u8")
	tmpFilename := filename + ".tmp"

	args = append(args,
		"-f", "hls",
		"-var_stream_map", varStreamMap,
		"-hls_time", "10",
		"-hls_segment_filename", "%v.ts",
		"-hls_segment_type", "fmp4",
		"-hls_flags", "append_list+single_file",
		"-hls_playlist_type", "event",
		"-master_pl_name", path.Base(tmpFilename),
		"%v.m3u8",
	)

	log.Printf("Run ffmpeg with args: %v\n", args)

	process := exec.Command("ffmpeg", args...)
	process.Dir = cwd

	process.Env = os.Environ()
	process.Stdout = os.Stdout
	process.Stderr = os.Stderr

	if err = process.Run(); err != nil {
		return
	}

	err = os.Rename(tmpFilename, filename)

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

func getStreamName(stream *ProbeStream) string {
	var parts []string
	if language, ok := stream.Tags["language"]; ok {
		parts = append(parts, language)
	}
	if title, ok := stream.Tags["title"]; ok {
		parts = append(parts, title)
	}
	return strings.Join(parts, " - ")
}

func getStreamsByType(streams []ProbeStream, codecType string) (results []*ProbeStream) {
	for i, s := range streams {
		if s.CodecType == codecType {
			results = append(results, &streams[i])
		}
	}
	return
}
