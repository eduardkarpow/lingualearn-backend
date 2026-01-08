package models

import (
	"time"

	"github.com/google/uuid"
)

type Test struct {
	Message string `json:"message"`
}

type Video struct {
	ID           uuid.UUID `json:"id"`
	Title        string    `json:"title"`
	Filename     string    `json:"filename"`
	Status       string    `json:"status"`
	Duration     *int      `json:"duration,omitempty"`
	ThumbnailKey *string   `json:"thumbnail_key,omitempty"`
	VideoKey     string    `json:"video_key,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type VideoResponse struct {
	ID           uuid.UUID `json:"id"`
	Title        string    `json:"title"`
	Status       string    `json:"status"`
	ThumbnailURL string    `json:"thumbnail_url"`
	StreamURL    string    `json:"stream_url"`
}
