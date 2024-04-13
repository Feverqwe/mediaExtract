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

func ProbeFile(inputIndex int, filename string) (result *ProbeResult, err error) {
	log.Printf("Probe file %s\n", filename)

	process := exec.Command("ffprobe", "-loglevel", "warning", "-hide_banner", "-i", filename, "-print_format", "json", "-show_format", "-show_streams")

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

	for _, stream := range result.Streams {
		stream.inputIndex = inputIndex
	}

	return
}

func FfmpegExtractStreams(cwd string, files []string, probeStreams []ProbeStream, options *Options) (err error) {
	streamIdx := 0
	getStreamIdx := func() (idx int) {
		idx = streamIdx
		streamIdx++
		return
	}

	hlsStreams, hlsArgs, err := getHlsArgs(getStreamIdx, probeStreams, options)
	if err != nil {
		return
	}

	subtitleStreams, subtitlesArgs, postProcessSubtitles, err := getSubtitleArgs(cwd, getStreamIdx, probeStreams, options)
	if err != nil {
		return
	}

	streams := append(hlsStreams, subtitleStreams...)

	usedFiles := getUsedFiles(files, streams)

	args := []string{"-hide_banner", "-loglevel", "warning", "-stats", "-y"}
	for _, fileaname := range usedFiles {
		args = append(args, "-i", fileaname)
	}

	args = append(args, hlsArgs...)
	args = append(args, subtitlesArgs...)

	mainPlName := "main.m3u8"
	if _, err = os.Stat(path.Join(cwd, mainPlName)); err == nil {
		log.Printf("Main playlist exists, skip\n")
		return
	}

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

	if err = postProcessSubtitles(); err != nil {
		return
	}

	err = buildMainPlaylist(cwd, streams, mainPlName)

	return
}

func getHlsArgs(getStreamIdx func() int, probeStreams []ProbeStream, options *Options) (streams []FloatStream, args []string, err error) {
	var hasHavc bool

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
			index:      index,
			inputIndex: fmt.Sprintf("%d:%d", stream.inputIndex, stream.Index),
			typeIndex:  fmt.Sprintf("v:%d", typeIndex),
			stream:     stream,
			format:     &format,
		})
	}
	if videoStreamIdx == 0 {
		err = fmt.Errorf("video streams is empty")
		return
	}

	var audioStreamIdx = 0
	for _, stream := range getStreamsByType(probeStreams, AUDIO_CODEC) {
		language := stream.Tags["language"]
		title := stream.Tags["title"]
		if len(options.aLangs) > 0 && !ArrayContain(options.aLangs, language) {
			continue
		}
		if len(options.aMasks) > 0 && !MatchMasksString(options.aMasks, title) {
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
			index:      index,
			inputIndex: fmt.Sprintf("%d:%d", stream.inputIndex, stream.Index),
			typeIndex:  fmt.Sprintf("a:%d", typeIndex),
			stream:     stream,
			format:     &format,
		})
	}
	if audioStreamIdx == 0 {
		err = fmt.Errorf("audio streams is empty")
		return
	}

	for _, stream := range streams {
		args = append(args, "-map", stream.inputIndex)
	}

	for _, stream := range streams {
		key := fmt.Sprintf("-b:%d", stream.index)
		args = append(args, key, stream.getBitrate())
	}

	for _, stream := range streams {
		codecKey := fmt.Sprintf("-codec:%d", stream.index)
		args = append(append(args, codecKey), stream.getCodecArgs()...)
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

	args = append(args, formatArgs...)

	return
}

func getSubtitleArgs(cwd string, getStreamIdx func() int, probeStreams []ProbeStream, options *Options) (streams []FloatStream, args []string, postProcess func() error, err error) {
	for _, stream := range getStreamsByType(probeStreams, SUBTITLE_CODEC) {
		if ArrayContain(SKIP_CODECS, stream.CodecName) {
			continue
		}

		language := stream.Tags["language"]
		title := stream.Tags["title"]
		if len(options.sLangs) > 0 && !ArrayContain(options.sLangs, language) {
			continue
		}
		if len(options.sMasks) > 0 && !MatchMasksString(options.sMasks, title) {
			continue
		}

		var format TargetFormat
		if format, err = getFormat(stream.CodecName); err != nil {
			return
		}

		index := getStreamIdx()
		typeIndex := len(streams)
		streams = append(streams, FloatStream{
			index:      index,
			inputIndex: fmt.Sprintf("%d:%d", stream.inputIndex, stream.Index),
			typeIndex:  fmt.Sprintf("s:%d", typeIndex),
			stream:     stream,
			format:     &format,
		})
	}

	for _, stream := range streams {
		args = append(args, "-map", stream.inputIndex)
		codecKey := fmt.Sprintf("-codec:%d", stream.index)
		args = append(args, codecKey)
		args = append(args, stream.getCodecArgs()...)
		format := stream.format.format
		args = append(args, "-f", format, stream.getChunkName())
	}

	postProcess = func() (err error) {
		for _, stream := range streams {
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
		return
	}

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
			name := f.getStreamName()
			lines = append(lines, fmt.Sprintf("#EXT-X-MEDIA:TYPE=AUDIO,NAME=\"%s\",URI=\"%s\"", name, url.QueryEscape(filename)))
		case SUBTITLE_CODEC:
			filename := path.Base(plFullName)
			name := f.getStreamName()
			lines = append(lines, fmt.Sprintf("#EXT-X-MEDIA:TYPE=SUBTITLES,NAME=\"%s\",URI=\"%s\"", name, url.QueryEscape(filename)))
		}
	}

	data := strings.Join(lines, "\n")
	err = os.WriteFile(filename, []byte(data), FILE_PERM)

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

func getUsedFiles(files []string, streams []FloatStream) (usedFiles []string) {
	usedInputs := make(map[int]bool)
	for i := range files {
		usedInputs[i] = false
	}
	for _, stream := range streams {
		usedInputs[stream.stream.inputIndex] = true
	}

	for i := range usedInputs {
		usedFiles = append(usedFiles, files[i])
	}
	return
}
