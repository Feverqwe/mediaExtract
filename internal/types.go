package internal

type ArrayFlags []string

func (i *ArrayFlags) String() string {
	return "<list>"
}

func (i *ArrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type BasicOptions struct {
	meta                  bool
	aLangs                ArrayFlags
	sLangs                ArrayFlags
	aMasks                ArrayFlags
	sMasks                ArrayFlags
	hlsSplitByTime        bool
	hlsTime               int
	hlsSegmentType        string
	hlsMasterPlaylistName string
}

type Options struct {
	BasicOptions
	meta   bool
	target string
}

func NewFileOptions(basicOptions BasicOptions, target string) Options {
	return Options{
		BasicOptions: basicOptions,
		target:       target,
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
