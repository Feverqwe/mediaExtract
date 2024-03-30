package internal

import "github.com/gobwas/glob"

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
