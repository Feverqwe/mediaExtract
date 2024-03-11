package internal

type Options struct {
	aLangs                []string
	sLangs                []string
	hlsSplitByTime        bool
	hlsTime               int
	hlsSegmentType        string
	hlsMasterPlaylistName string
}

func NewOptions(aLangs []string, sLangs []string, hlsSplitByTime bool, hlsTime int, hlsSegmentType string, hlsMasterPlaylistName string) Options {
	return Options{
		aLangs:                aLangs,
		sLangs:                sLangs,
		hlsSplitByTime:        hlsSplitByTime,
		hlsTime:               hlsTime,
		hlsSegmentType:        hlsSegmentType,
		hlsMasterPlaylistName: hlsMasterPlaylistName,
	}
}
