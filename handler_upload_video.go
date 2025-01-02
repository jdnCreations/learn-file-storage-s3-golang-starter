package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	maxMemory := 1 << 30
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxMemory))

	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading video", videoID, "by user", userID)

	videoData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Could not find metadata for video", err)
	}

	if videoData.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "You are unauthorized", err)
		return
	}

	mpFile, mpHeader, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, 422, "Error getting data from video FormFile", err)
		return
	}
	defer mpFile.Close()
	mediaType := mpHeader.Header.Get("Content-Type")

	fileType, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		respondWithError(w, 422, "Couldn't get extension type from Content-Type", err)
		return
	}

	if fileType != "video/mp4" {
		respondWithError(w, 422, "File must be in .mp4 format", err)
		return
	}

	slicey := make([]byte, 32)
	_, err = rand.Read(slicey)
	if err != nil {
		respondWithError(w, 400, "Could not create a random thing", err)
	}
	base64String := base64.RawURLEncoding.EncodeToString(slicey)	


	tmpFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, 422, "Could not create temp file", err)
		return
	}

  _, err = io.Copy(tmpFile, mpFile)
	if err != nil {
		respondWithError(w, 422, "Could not copy file", err)
		return
	}

  newPath, err := processVideoForFastStart(tmpFile.Name())
  if err != nil {
    respondWithError(w, 422, "Could not process video for fast start", err)
    return
  }

  processedFile, err := os.Open(newPath)
  if err != nil {
    respondWithError(w, 422, "Could not open file", err)
    return
  }
  
  defer processedFile.Close()
  defer os.Remove(processedFile.Name())

	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	ratio, err := getVideoAspectRatio(tmpFile.Name())
	if err != nil {
		respondWithError(w, 422, "Could not get aspect ratio from video", err)
		return
	}

  var prefix string
  switch ratio {
  case "16:9":
    prefix = "landscape/"
  case "9:16":
    prefix = "portrait/"
  default:
    prefix = "other/"
  }

	key := prefix + base64String + ".mp4"

	// reset pointer
	tmpFile.Seek(0, io.SeekStart)
	_, err = cfg.s3Client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: &cfg.s3Bucket,
		Key: &key,
		Body: processedFile,
		ContentType: &fileType,
	})

	if err != nil {
		respondWithError(w, 422, "Failed to upload object to S3", err)
		return
	}
	
	videoURL := fmt.Sprintf("%s,%s", cfg.s3Bucket, key)
	videoData.VideoURL = &videoURL 

	err = cfg.db.UpdateVideo(videoData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update video URL in the database", err)
		return
	}

  video, err := cfg.dbVideoToSignedVideo(videoData)
  if err != nil {
    respondWithError(w, 422, "Could not convert to signed video", err)
    return
  }

	respondWithJSON(w, http.StatusOK, video)
}
