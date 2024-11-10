package internal

import (
	"fmt"
	"strings"
)

type FloatStream struct {
	index      int
	inputIndex string
	typeIndex  string
	stream     *ProbeStream
	format     *TargetFormat
}

func (s *FloatStream) getChunkName() string {
	return fmt.Sprintf("%d.%s", s.index, s.format.ext)
}

func (s *FloatStream) getPlaylistName() string {
	return fmt.Sprintf("%d.m3u8", s.index)
}

func (s *FloatStream) getStreamName() string {
	var parts []string
	if language, ok := s.stream.Tags["language"]; ok {
		parts = append(parts, language)
	}
	if title, ok := s.stream.Tags["title"]; ok {
		parts = append(parts, title)
	}
	if len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%d-%d", s.stream.inputIndex, s.stream.Index))
	}
	return strings.Join(parts, " - ")
}

func (s *FloatStream) getCodecArgs() (codecArgs []string) {
	codecArgs = append(codecArgs, s.format.codec)
	if len(s.format.codecParams) > 0 {
		codecArgs = append(codecArgs, s.format.codecParams...)
	}
	return
}

func (s *FloatStream) getBitrate() (bitrate string) {
	bitrate = s.stream.BitRate
	if bps, ok := s.stream.Tags["BPS"]; ok {
		bitrate = bps
	}
	if bitrate == "" {
		bitrate = "1"
	}
	return
}
