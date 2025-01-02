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

// func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
//   presignClient := s3.NewPresignClient(s3Client)
//   req, err := presignClient.PresignGetObject(context.Background(), &s3.GetObjectInput{
//     Bucket: &bucket,
//     Key: &key,
//   },
//   s3.WithPresignExpires(expireTime))
//   if err != nil {
//     return "", err
//   }

//   return req.URL, nil
// }

// func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
//   fmt.Printf("Processing video: %+v\n", video)
//   if video.VideoURL == nil {
//     fmt.Println("VideoURL is nil") // Add this
//     return video, nil 
//   }
//   fmt.Printf("Video URL value: %s\n", *video.VideoURL) // Add this
//   parts := strings.Split(*video.VideoURL, ",")
//   if len(parts) != 2 {
//     return database.Video{}, fmt.Errorf("invalid video URL format")
//   }
//   bucket := parts[0]
//   key := parts[1]
//   fmt.Printf("Bucket: %s, Key: %s\n", bucket, key) // Add this
//   url, err := generatePresignedURL(cfg.s3Client, bucket, key, 15 * time.Minute)
//   if err != nil {
//     return database.Video{}, err
//   }
//   video.VideoURL = &url

//   return video, nil
// }