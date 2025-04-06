package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
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

	videoMetadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Not able to get the video", err)
		return
	}

	if videoMetadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Not authorized to upload the thumbnail", nil)
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to extract thumbnail from form", err)
		return
	}
	defer file.Close()

	fileType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse media type", err)
		return
	}
	extension := strings.Split(fileType, "/")[1]

	if extension != "mp4" {
		respondWithError(w, http.StatusBadRequest, "Invalid file type", nil)
		return
	}

	tmpFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to create temp file", err)
		return
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	io.Copy(tmpFile, file)

  fastStatPath, err := cfg.processVideoForFastStart(tmpFile.Name())
  defer os.Remove(fastStatPath)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to fast start temp file", err)
		return
	}

  tmpFile, err = os.Open(fastStatPath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to create temp file", err)
		return
	}

	tmpFile.Seek(0, io.SeekStart)

	reader := make([]byte, 32)
	_, err = rand.Read(reader)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to generate random string", err)
		return
	}

	encodedName := base64.RawURLEncoding.EncodeToString(reader)
	aspect, err := cfg.getVideoAspectRatio(tmpFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to upload video", err)
		return
	}

	filename := fmt.Sprintf("%v/%v.%v", aspect, encodedName, extension)

	_, err = cfg.s3Client.PutObject(context.Background(),
		&s3.PutObjectInput{
			Bucket:      &cfg.s3Bucket,
			Body:        tmpFile,
			Key:         &filename,
			ContentType: &fileType,
		},
	)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to upload video", err)
		return
	}

	videoUrl := fmt.Sprintf("https://%v.s3.%v.amazonaws.com/%v", cfg.s3Bucket, cfg.s3Region, filename)
	videoMetadata.VideoURL = &videoUrl
	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoMetadata)
}
