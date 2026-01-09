package services

import (
	"context"
	"database/sql"
	"linguaLearn/internal/models"

	"github.com/google/uuid"
)

type VideoService struct {
	db *sql.DB
}

func NewVideoService(db *sql.DB) *VideoService {
	return &VideoService{db: db}
}

func (vs *VideoService) CreateVideo(ctx context.Context, video models.Video) error {
	query := `
        INSERT INTO videos (id, title, filename, status, video_key, created_at, updated_at) 
        VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
    `
	_, err := vs.db.ExecContext(ctx, query, video.ID, video.Title, video.Filename, video.Status, video.VideoKey)
	return err
}

func (vs *VideoService) UpdateStatus(ctx context.Context, videoId uuid.UUID, status string) error {
	query := `
		UPDATE videos SET status=$1 WHERE id=$2
	`
	_, err := vs.db.ExecContext(ctx, query, status, videoId)
	return err
}

func (vs *VideoService) UpdateAddInfo(ctx context.Context, videoId uuid.UUID, thumbnail_key string, duration int) error {
	query := `
		UPDATE videos SET thumbnail_key=$1, duration=$2 WHERE id=$3
	`
	_, err := vs.db.ExecContext(ctx, query, thumbnail_key, duration, videoId)
	return err
}

func (vs *VideoService) ListVideos(ctx context.Context) ([]models.Video, error) {
	rows, err := vs.db.QueryContext(ctx, "SELECT * FROM videos WHERE status='ready' ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var videos []models.Video

	for rows.Next() {
		var v models.Video
		// Scan the columns into the struct fields
		// Note: The order must match the SELECT statement exactly
		err := rows.Scan(&v.ID, &v.Title, &v.Filename, &v.Status, &v.Duration, &v.ThumbnailKey, &v.VideoKey, &v.CreatedAt, &v.UpdatedAt)
		if err != nil {
			return nil, err
		}
		videos = append(videos, v)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return videos, err
}

func (vs *VideoService) GetVideoByID(ctx context.Context, id string) (*models.Video, error) {
	video := &models.Video{}
	err := vs.db.QueryRowContext(ctx, "SELECT * FROM videos WHERE id = $1", id).Scan(
		&video.ID, &video.Title, &video.Filename, &video.Status,
		&video.Duration, &video.ThumbnailKey, &video.VideoKey,
		&video.CreatedAt, &video.UpdatedAt,
	)
	return video, err
}

func (vs *VideoService) CreateSub(ctx context.Context, videoId string, subKey string) error {
	query := `
		INSERT INTO subtitles (id, video_id, subtitle_key, shift) VALUES ($1, $2, $3, $4)
	`
	_, err := vs.db.ExecContext(ctx, query, uuid.New(), videoId, subKey, 0)
	return err
}
