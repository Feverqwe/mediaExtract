package internal

const MAIN_PLAYLIST_NAME = "main.m3u8"

const DIR_PERM = 0700
const FILE_PERM = 0600

const VIDEO_CODEC = "video"
const AUDIO_CODEC = "audio"
const SUBTITLE_CODEC = "subtitle"

var SKIP_CODECS = []string{"mjpeg", "hdmv_pgs_subtitle"}

var CODEC_TARGET_FORMAT = []TargetFormat{
	{
		codecNames: []string{"h264", "hevc"},
		codec:      "copy",
	}, {
		codecNames: []string{"mp3", "aac"},
		codec:      "copy",
	}, {
		codecNames:  []string{"ac3", "eac3", "dts", "truehd"},
		codec:       "libfdk_aac",
		codecParams: []string{"-vbr", "5"},
	}, {
		codecNames: []string{"subrip"},
		codec:      "webvtt",
		format:     "webvtt",
		ext:        "vtt",
	}, {
		codecNames:  []string{"mpeg4"},
		codec:       "h264",
		codecParams: []string{"-crf", "12", "-preset", "slow"},
	},
}
