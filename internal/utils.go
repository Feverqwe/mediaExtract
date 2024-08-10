package internal

import (
	"flag"
	"path"
	"strings"

	"github.com/gobwas/glob"
)

func ArrayContain(arr []string, value string) bool {
	for _, v := range arr {
		if v == value {
			return true
		}
	}
	return false
}

func MatchMasksString(masks []string, title string) bool {
	var g glob.Glob
	for _, m := range masks {
		g = glob.MustCompile(m)
		if g.Match(title) {
			return true
		}
	}
	return false
}

func GetBasicOptions(f *flag.FlagSet) BasicOptions {
	basicOptions := BasicOptions{}
	f.BoolVar(&basicOptions.meta, "meta", false, "Show metadata")
	f.Var(&basicOptions.aLangs, "al", "Add audio language filter")
	f.Var(&basicOptions.sLangs, "sl", "Add subtilte language filter")
	f.Var(&basicOptions.aMasks, "aMask", "Add audio title mask filter")
	f.Var(&basicOptions.sMasks, "sMask", "Add subtilte title mask filter")
	f.IntVar(&basicOptions.hlsTime, "hlsTime", 10, "Set hls segment time")
	f.BoolVar(&basicOptions.hlsSplitByTime, "hlsSplitByTime", false, "Add hls split by time flag")
	f.StringVar(&basicOptions.hlsSegmentType, "hlsSegmentType", "", "Force set hls segment type: mpegts or fmp4")
	f.StringVar(&basicOptions.hlsMasterPlaylistName, "hlsMasterPlaylistName", "", "Create HLS master playlist with the given name")
	return basicOptions
}

func GetTargetName(fp string) string {
	filename := path.Base(fp)
	name := strings.TrimSuffix(filename, path.Ext(filename))
	placeName := name + ".media"
	return path.Join(path.Dir(fp), placeName)
}
