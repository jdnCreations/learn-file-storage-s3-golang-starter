package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
)

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	var b bytes.Buffer
  cmd.Stdout = &b
	err := cmd.Run()
  if err != nil {
    return "", err
  }

	type FFProbeOutput struct {
    Streams []struct {
        Width  int `json:"width"`
        Height int `json:"height"`
    } `json:"streams"`
	}

	probeData := FFProbeOutput{}

	err = json.Unmarshal(b.Bytes(), &probeData)
	if err != nil {
		return "", err
	}

	width := probeData.Streams[0].Width
	height := probeData.Streams[0].Height

	ratio := float64(width) / float64(height)
  fmt.Printf("Debug - Width: %d, Height: %d, Ratio: %f\n", width, height, ratio)

  if math.Abs(ratio - 1.778) < 0.01 {
    fmt.Println("Debug - Detected as 16:9")
    return "16:9", nil
  }

  if math.Abs(ratio - 0.5625) < 0.01 {
    fmt.Println("Debug - Detected as 9:16")
    return "9:16", nil
  }

  fmt.Println("Debug - Detected as other")
	return "other", nil 
}

func processVideoForFastStart(filePath string) (string, error) {
  updatedPath := filePath + ".processing"
  cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", updatedPath)
  var b bytes.Buffer
  cmd.Stdout = &b
	err := cmd.Run()
  if err != nil {
    return "", err
  }
  return updatedPath, nil
}