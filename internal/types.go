package internal

type Options struct {
	aLangs                []string
	sLangs                []string
	aMasks                []string
	sMasks                []string
	hlsSplitByTime        bool
	hlsTime               int
	hlsSegmentType        string
	hlsMasterPlaylistName string
	meta                  bool
	target                string
}

func NewOptions(aLangs []string, sLangs []string, aMasks []string, sMasks []string, hlsSplitByTime bool, hlsTime int, hlsSegmentType string, hlsMasterPlaylistName string, meta bool, target string) Options {
	return Options{
		aLangs:                aLangs,
		sLangs:                sLangs,
		aMasks:                aMasks,
		sMasks:                sMasks,
		hlsSplitByTime:        hlsSplitByTime,
		hlsTime:               hlsTime,
		hlsSegmentType:        hlsSegmentType,
		hlsMasterPlaylistName: hlsMasterPlaylistName,
		meta:                  meta,
		target:                target,
	}
}

type TargetFormat struct {
	codecNames  []string
	codec       string
	codecParams []string
	format      string
	ext         string
}
type ProbeStream struct {
	inputIndex    int
	Index         int               `json:"index"`
	CodecName     string            `json:"codec_name"`
	CodecType     string            `json:"codec_type"`
	Tags          map[string]string `json:"tags"`
	ChannelLayout string            `json:"channel_layout"`
	BitRate       string            `json:"bit_rate"`
}

type ProbeFormat struct {
	FormatName string            `json:"format_name"`
	Tags       map[string]string `json:"tags"`
}

type ProbeResult struct {
	Streams []ProbeStream `json:"streams"`
	Format  ProbeFormat   `json:"format"`
}
