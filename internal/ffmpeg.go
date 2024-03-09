package internal

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
)

type TargetFormat struct {
	codec        string
	codecParams  []string
	format       string
	formatParams []string
	ext          string
	configurate  func(format TargetFormat, stream *ProbeStream) TargetFormat
}

var CODEC_TARGET_FORMAT = map[string]TargetFormat{
	"h264": {
		codec:  "copy",
		format: "h264",
		ext:    "m4v",
	},
	"subrip": {
		codec:  "",
		format: "webvtt",
		ext:    "vtt",
	},
	"ac3": {
		codec:       "libopus",
		codecParams: []string{"-vbr", "on", "-application", "audio", "-compression_level", "10"},
		format:      "opus",
		ext:         "opus",
		configurate: func(format TargetFormat, stream *ProbeStream) TargetFormat {
			if stream.CodecName == "ac3" {
				if stream.ChannelLayout == "5.1(side)" {
					format.codecParams = append(format.codecParams, "-af", "channelmap=channel_layout=5.1")
				}
				if stream.ChannelLayout == "5.0(side)" {
					format.codecParams = append(format.codecParams, "-af", "channelmap=channel_layout=5.0")
				}
			}
			return format
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

	var format = TargetFormat{
		ext:    stream.CodecName,
		format: stream.CodecName,
		codec:  "copy",
	}
	if val, ok := CODEC_TARGET_FORMAT[stream.CodecName]; ok {
		if val.configurate != nil {
			format = val.configurate(val, stream)
		} else {
			format = val
		}
		stream.OrigCodecName = stream.CodecName
		stream.CodecName = format.format
	}

	name := strconv.Itoa(stream.Index) + "." + format.ext
	filename = path.Join(cwd, name)

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

	if err = os.Rename(tmpFilename, filename); err != nil {
		return
	}

	return
}
