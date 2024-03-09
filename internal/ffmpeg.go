package internal

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
)

type ProbeStream struct {
	Index     int               `json:"index"`
	CodecName string            `json:"codec_name"`
	CodecType string            `json:"codec_type"`
	Tags      map[string]string `json:"tags"`
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

	process := exec.Command("ffprobe", "-i", filepath, "-print_format", "json", "-show_format", "-show_streams")

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

func FfmpegExtractStream(cwd string, filepath string, stream ProbeStream) (filename string, err error) {
	log.Printf("Extract stream %d %s\n", stream.Index, filepath)

	name := strconv.Itoa(stream.Index) + "." + stream.CodecName
	filename = path.Join(cwd, name)

	if _, err = os.Stat(path.Join(cwd, filename)); err == nil {
		log.Printf("File exists, skip extracting\n")
		return
	}

	tmpFilename := filename + ".tmp"

	process := exec.Command("ffmpeg", "-y", "-i", filepath, "-map", "0:"+strconv.Itoa(stream.Index), "-c", "copy", "-f", stream.CodecName, tmpFilename)
	process.Dir = cwd

	process.Env = os.Environ()
	process.Stdin = os.Stdin
	process.Stdout = os.Stdout
	process.Stderr = os.Stderr

	if err = process.Run(); err != nil {
		return
	}

	if err = os.Rename(tmpFilename, filename); err != nil {
		return
	}

	return
}
