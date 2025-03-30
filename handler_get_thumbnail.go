package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerThumbnailGet(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid video ID", err)
		return
	}

	tn, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Thumbnail not found", err)
		return
	}

  file, err := os.ReadFile(*tn.ThumbnailURL)
  if err != nil {
    fmt.Printf("unable to read file: %v\n", err)
  }

  contentType := "image/png"

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(file)))

	_, err = w.Write(file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error writing response", err)
		return
	}
}
