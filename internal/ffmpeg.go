package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
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
		codecNames:  []string{"ac3", "eac3", "dts"},
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
	index        int
	typeIndex    string
	stream       *ProbeStream
	targetFormat *TargetFormat
}

func (s *FloatStream) getChunkName() string {
	return fmt.Sprintf("%d.%s", s.index, s.targetFormat.ext)
}

func (s *FloatStream) getPlaylistName() string {
	return fmt.Sprintf("%d.m3u8", s.index)
}

func FfmpegExtractStreams(cwd, filepath string, probeStreams []ProbeStream, options Options) (err error) {
	const INPUT_INDEX = 0
	var streams []FloatStream
	var hasHavc bool

	streamIdx := 0
	getStreamIdx := func() (idx int) {
		idx = streamIdx
		streamIdx++
		return
	}

	var videoStreamIdx = 0
	for _, stream := range getStreamsByType(probeStreams, VIDEO_CODEC) {
		if ArrayContain(SKIP_CODECS, stream.CodecName) {
			continue
		}

		var format TargetFormat
		if format, err = getFormat(stream.CodecName); err != nil {
			return
		}

		if stream.CodecName == "hevc" {
			hasHavc = true
		}

		index := getStreamIdx()
		typeIndex := videoStreamIdx
		videoStreamIdx++
		streams = append(streams, FloatStream{
			index:        index,
			typeIndex:    fmt.Sprintf("v:%d", typeIndex),
			stream:       stream,
			targetFormat: &format,
		})
	}
	if videoStreamIdx == 0 {
		err = fmt.Errorf("video streams is empty")
		return
	}

	var audioStreamIdx = 0
	for _, stream := range getStreamsByType(probeStreams, AUDIO_CODEC) {
		language := stream.Tags["language"]
		if len(options.aLangs) > 0 && !ArrayContain(options.aLangs, language) {
			continue
		}

		var format TargetFormat
		if format, err = getFormat(stream.CodecName); err != nil {
			return
		}

		index := getStreamIdx()
		typeIndex := audioStreamIdx
		audioStreamIdx++
		streams = append(streams, FloatStream{
			index:        index,
			typeIndex:    fmt.Sprintf("a:%d", typeIndex),
			stream:       stream,
			targetFormat: &format,
		})
	}
	if audioStreamIdx == 0 {
		err = fmt.Errorf("audio streams is empty")
		return
	}

	var subtitleStreams []FloatStream
	for _, stream := range getStreamsByType(probeStreams, SUBTITLE_CODEC) {
		if ArrayContain(SKIP_CODECS, stream.CodecName) {
			continue
		}

		language := stream.Tags["language"]
		if len(options.sLangs) > 0 && !ArrayContain(options.sLangs, language) {
			continue
		}

		var format TargetFormat
		if format, err = getFormat(stream.CodecName); err != nil {
			return
		}

		index := getStreamIdx()
		typeIndex := len(subtitleStreams)
		subtitleStreams = append(subtitleStreams, FloatStream{
			index:        index,
			typeIndex:    fmt.Sprintf("s:%d", typeIndex),
			stream:       stream,
			targetFormat: &format,
		})
	}

	mainPlName := "main.m3u8"
	if _, err = os.Stat(path.Join(cwd, mainPlName)); err == nil {
		log.Printf("Main playlist exists, skip\n")
		return
	}

	var hlsArgs []string

	for _, stream := range streams {
		mapVal := fmt.Sprintf("%d:%d", INPUT_INDEX, stream.stream.Index)
		hlsArgs = append(hlsArgs, "-map", mapVal)
	}

	for _, stream := range streams {
		key := fmt.Sprintf("-b:%d", stream.index)
		hlsArgs = append(hlsArgs, key, "1")
	}

	for _, stream := range streams {
		codecKey := fmt.Sprintf("-codec:%d", stream.index)
		codecArgs := getCodecArgs(*stream.targetFormat)
		hlsArgs = append(append(hlsArgs, codecKey), codecArgs...)
	}

	var varStreamMapItems []string
	for _, stream := range streams {
		val := fmt.Sprintf("%s,agroup:main", stream.typeIndex)
		varStreamMapItems = append(varStreamMapItems, val)
	}
	varStreamMap := strings.Join(varStreamMapItems, " ")

	segmentType := "mpegts"
	hlsFlags := []string{"append_list", "single_file"}
	if hasHavc {
		segmentType = "fmp4"
	}
	if options.hlsSegmentType != "" {
		segmentType = options.hlsSegmentType
	}
	if options.hlsSplitByTime {
		hlsFlags = append(hlsFlags, "split_by_time")
	}

	formatArgs := []string{
		"-f", "hls",
		"-var_stream_map", varStreamMap,
		"-hls_time", fmt.Sprintf("%d", options.hlsTime),
		"-hls_segment_filename", "%v.ts",
		"-hls_segment_type", segmentType,
		"-hls_flags", strings.Join(hlsFlags, "+"),
		"-hls_playlist_type", "event",
	}

	if options.hlsMasterPlaylistName != "" {
		formatArgs = append(formatArgs, "-master_pl_name", options.hlsMasterPlaylistName)
	}

	formatArgs = append(formatArgs, "%v.m3u8")

	hlsArgs = append(hlsArgs, formatArgs...)

	var subtitlesArgs []string
	for _, stream := range subtitleStreams {
		mapVal := fmt.Sprintf("%d:%d", INPUT_INDEX, stream.stream.Index)
		subtitlesArgs = append(subtitlesArgs, "-map", mapVal)
		codecKey := fmt.Sprintf("-codec:%d", stream.index)
		subtitlesArgs = append(subtitlesArgs, codecKey)
		codecArgs := getCodecArgs(*stream.targetFormat)
		subtitlesArgs = append(subtitlesArgs, codecArgs...)
		format := stream.targetFormat.format
		subtitlesArgs = append(subtitlesArgs, "-f", format, stream.getChunkName())
	}

	args := []string{"-hide_banner", "-y", "-i", filepath}
	args = append(args, hlsArgs...)
	args = append(args, subtitlesArgs...)

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

	for _, stream := range subtitleStreams {
		plFilename := path.Join(cwd, stream.getPlaylistName())

		data := strings.Join([]string{
			"#EXTM3U",
			"#EXT-X-TARGETDURATION:0",
			"#EXT-X-PLAYLIST-TYPE:VOD",
			stream.getChunkName(),
			"#EXT-X-ENDLIST",
		}, "\n")

		if err = os.WriteFile(plFilename, []byte(data), FILE_PERM); err != nil {
			return
		}
	}

	err = buildMainPlaylist(cwd, append(streams, subtitleStreams...), mainPlName)

	return
}

func buildMainPlaylist(cwd string, streams []FloatStream, name string) (err error) {
	filename := path.Join(cwd, name)

	var lines = []string{
		"#EXTM3U",
	}

	for _, f := range streams {
		plFullName := path.Join(cwd, f.getPlaylistName())
		switch f.stream.CodecType {
		case VIDEO_CODEC:
			filename := path.Base(plFullName)
			lines = append(lines, "#EXT-X-STREAM-INF:PROGRAM-ID=1", filename)
		case AUDIO_CODEC:
			filename := path.Base(plFullName)
			name := getStreamName(f.stream)
			lines = append(lines, fmt.Sprintf("#EXT-X-MEDIA:TYPE=AUDIO,NAME=\"%s\",URI=\"%s\"", name, url.QueryEscape(filename)))
		case SUBTITLE_CODEC:
			filename := path.Base(plFullName)
			name := getStreamName(f.stream)
			lines = append(lines, fmt.Sprintf("#EXT-X-MEDIA:TYPE=SUBTITLES,NAME=\"%s\",URI=\"%s\"", name, url.QueryEscape(filename)))
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

func getCodecArgs(format TargetFormat) (codecArgs []string) {
	codecArgs = append(codecArgs, format.codec)
	if len(format.codecParams) > 0 {
		codecArgs = append(codecArgs, format.codecParams...)
	}
	return
}

func getFormat(codecName string) (format TargetFormat, err error) {
	for _, f := range CODEC_TARGET_FORMAT {
		for _, c := range f.codecNames {
			if c == codecName {
				format = f
				return
			}
		}
	}
	err = fmt.Errorf("unsupported codec: %s", codecName)
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
