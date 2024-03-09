package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
)

type TargetFormat struct {
	codecNames   []string
	codec        string
	codecParams  []string
	format       string
	formatParams []string
	ext          string
	configurate  func(cwd string, format TargetFormat, stream *ProbeStream) TargetFormat
	postProcess  func(cwd string, filename string) (string, error)
}

func hlsConfigure(cwd string, format TargetFormat, stream *ProbeStream, ext string) TargetFormat {
	idxStr := strconv.Itoa(stream.Index)
	sigName := idxStr + "-sig"

	format.formatParams = append(format.formatParams, "-hls_segment_filename", sigName+ext)
	return format
}

var CODEC_TARGET_FORMAT = []TargetFormat{
	{
		codecNames: []string{"h264", "hevc"},
		codec:      "copy",
		format:     "hls",
		formatParams: []string{
			"-hls_time", "10",
			"-hls_segment_filename", "sig.ts",
			"-hls_flags", "append_list+single_file",
			"-hls_playlist_type", "event",
		},
		ext: "m3u8",
		configurate: func(cwd string, format TargetFormat, stream *ProbeStream) TargetFormat {
			return hlsConfigure(cwd, format, stream, ".m4v")
		},
	},
	{
		codecNames: []string{"subrip"},
		format:     "webvtt",
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
		},
	}, {
		codecNames:  []string{"ac3", "eac3"},
		codec:       "libfdk_aac",
		codecParams: []string{"-vbr", "5"},
		format:      "hls",
		formatParams: []string{
			"-hls_time", "10",
			"-hls_segment_filename", "sig.opus",
			"-hls_flags", "append_list+single_file",
			"-hls_playlist_type", "event",
		},
		ext: "m3u8",
		configurate: func(cwd string, format TargetFormat, stream *ProbeStream) TargetFormat {
			return hlsConfigure(cwd, format, stream, ".m4a")
		},
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

func FfmpegExtractStream(cwd string, filepath string, stream *ProbeStream) (filename string, err error) {
	log.Printf("Extract stream %d %s\n", stream.Index, filepath)

	if stream.CodecType != VIDEO_CODEC && stream.CodecType != AUDIO_CODEC && stream.CodecType != SUBTITLE_CODEC {
		log.Printf("Codec type is not supported: %s, skip\n", stream.CodecType)
		return
	}

	format, ok := getFormat(stream.CodecName)
	if !ok {
		panic(fmt.Errorf("unsupported codec: %s", stream.CodecName))
	}

	if format.configurate != nil {
		format = format.configurate(cwd, format, stream)
	}
	stream.OrigCodecName = stream.CodecName
	stream.CodecName = format.format

	name := strconv.Itoa(stream.Index) + "." + format.ext
	filename = path.Join(cwd, name)

	defer (func() {
		if err == nil && format.postProcess != nil {
			filename, err = format.postProcess(cwd, filename)
		}
	})()

	if _, err = os.Stat(filename); err == nil {
		log.Printf("File exists, skip extracting\n")
		return
	}

	tmpFilename := filename + ".tmp"

	args := []string{"-hide_banner", "-y", "-i", filepath, "-map", "0:" + strconv.Itoa(stream.Index)}

	if len(format.codec) > 0 {
		args = append(args, "-c", format.codec)
	}
	if len(format.codecParams) > 0 {
		args = append(args, format.codecParams...)
	}

	args = append(args, "-f", format.format)
	if len(format.formatParams) > 0 {
		args = append(args, format.formatParams...)
	}

	args = append(args, tmpFilename)

	log.Printf("Run ffmpeg with args: %v\n", args)

	process := exec.Command("ffmpeg", args...)
	process.Dir = cwd

	process.Env = os.Environ()
	process.Stdin = os.Stdin
	process.Stdout = os.Stdout
	process.Stderr = os.Stderr

	if err = process.Run(); err != nil {
		return
	}

	err = os.Rename(tmpFilename, filename)

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
