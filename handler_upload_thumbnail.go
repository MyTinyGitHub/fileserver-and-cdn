package main

import (
	"fmt"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to extract thumbnail from form", err)
		return
	}
	defer file.Close()

	thumbStruct := thumbnail{
		data:      nil,
		mediaType: header.Header.Get("Content-Type"),
	}

	videoMetadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Not able to get the video", err)
		return
	}

	if videoMetadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Not authorized to upload the thumbnail", nil)
		return
	}

	fileType, _, err := mime.ParseMediaType(thumbStruct.mediaType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse media type", err)
		return
	}
	extension := strings.Split(fileType, "/")[1]

	if extension != "jpg" && extension != "png" {
		respondWithError(w, http.StatusBadRequest, "Invalid file type", nil)
		return
	}

	filename := fmt.Sprintf("%v.%v", videoID.String(), extension)
	filePath := filepath.Join(cfg.assetsRoot, filename)

	fileWriter, _ := os.Create(filePath)
	io.Copy(fileWriter, file)
	defer fileWriter.Close()

	thumbnailURl := fmt.Sprintf("http://localhost:8091/assets/%v", filename)
	videoMetadata.ThumbnailURL = &thumbnailURl

	cfg.db.UpdateVideo(videoMetadata)

	respondWithJSON(w, http.StatusOK, videoMetadata)
}
