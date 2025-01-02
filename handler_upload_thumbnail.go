package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
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


	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// implement the upload here
	const maxMemory = 10 << 20
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, 422, "Couldn't parse some data", err)
		return
	}

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, 422, "Error getting data from thumbnail", err)
		return
	}

	mediaType := header.Header.Get("Content-Type")
	fmt.Printf("Content-Type: %s\n", mediaType)

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, 422, "Error getting mediaType", err)
		return
	}

	// save bytes to file at path /assets/<videoID>.<file_extension>
	fileType, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		respondWithError(w, 422, "Couldn't get extension type from Content-Type", err)
		return
	}
	if fileType != "image/jpeg" && fileType != "image/png" {
		respondWithError(w, 422, "Incorrect file format, must be .jpeg, or .png", err)
		return
	}
	ext, err := mime.ExtensionsByType(fileType)
	if err != nil {
		respondWithError(w, 422, "Couldn't get extension type from Content-Type", err)
		return
	}
	if len(ext) == 0 {
		respondWithError(w, 400, "Invalid file format", err)
		return
	}

	slicey := make([]byte, 32)
	_, err = rand.Read(slicey)
	if err != nil {
		respondWithError(w, 400, "Could not create a random thing", err)
	}
	base64String := base64.RawURLEncoding.EncodeToString(slicey)	
	filename := base64String + ext[0] 
	fp := filepath.Join(cfg.assetsRoot, filename)
	newFile, err := os.Create(fp)
	if err != nil {
		respondWithError(w, 422, "Error creating new file", err)
		return
	}
	defer newFile.Close()

	_, err = io.Copy(newFile, file)
	if err != nil {
		respondWithError(w, 422, "Could not copy file", err)
		return
	}

	thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, filename)

	video.ThumbnailURL = &thumbnailURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, 422, "Could not update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
