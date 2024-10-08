package internal

import (
	"flag"
	"log"
	"os"
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
	return basicOptions
}

func GetTargetName(fp string, td string) string {
	filename := path.Base(fp)
	name := strings.TrimSuffix(filename, path.Ext(filename))
	placeName := name + ".media"
	if td == "" {
		td = path.Dir(fp)
	}
	return path.Join(td, placeName)
}

func ClenupTargetFolder(cwd string, streams []FloatStream, extraFiles []string) (err error) {
	files := extraFiles

	for _, stream := range streams {
		files = append(files, stream.getPlaylistName())
	}

	for _, relFilename := range files {
		filename := path.Join(cwd, relFilename)
		if _, sErr := os.Stat(filename); sErr == nil {
			log.Printf("Removing incomplete playlist \"%s\"\n", relFilename)
			if err = os.Remove(filename); err != nil {
				return
			}
		}
	}

	return
}
