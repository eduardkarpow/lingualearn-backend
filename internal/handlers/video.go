package handlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"linguaLearn/internal/models"
	"linguaLearn/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handlers struct {
	videoSvc *services.VideoService
	storage  *services.StorageService
}

type AddInfoResponse struct {
	duration int
	pr       io.Reader
}

func NewHandlers(videoSvc *services.VideoService, storage *services.StorageService) *Handlers {
	return &Handlers{videoSvc: videoSvc, storage: storage}
}

func (h *Handlers) TestFiber(c *fiber.Ctx) error {
	message := "asdas"
	return c.JSON(models.Test{
		Message: message,
	})
}

// POST /api/v1/videos - Upload video file
func (h *Handlers) UploadVideo(c *fiber.Ctx) error {
	file, err := c.FormFile("video")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "No video file provided"})
	}

	// Generate unique filename
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%s%s", uuid.New().String(), ext)

	// Open and read file
	f, err := file.Open()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to read file"})
	}
	defer f.Close()

	fileBytes, err := io.ReadAll(f)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to read file"})
	}

	// Save to MinIO
	videoKey, err := h.storage.SaveFile(c.Context(), filename, file.Header.Get("Content-Type"), fileBytes)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to save video"})
	}

	// Create video record
	video := models.Video{
		ID:       uuid.New(),
		Title:    file.Filename,
		Filename: filename,
		Status:   "processing",
		VideoKey: videoKey,
	}

	// Save to DB
	if err := h.videoSvc.CreateVideo(c.Context(), video); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to save metadata"})
	}

	// Start background processing
	go h.processVideoAsync(c.Context(), video.ID)

	return c.JSON(models.VideoResponse{
		ID:        video.ID,
		Title:     video.Title,
		Status:    video.Status,
		StreamURL: h.storage.GetObjectURL(videoKey),
	})
}

// GET /api/v1/videos - List videos
func (h *Handlers) ListVideos(c *fiber.Ctx) error {
	videos, err := h.videoSvc.ListVideos(c.Context())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("Failed to fetch videos %e", err)})
	}

	var response []models.VideoResponse
	for _, v := range videos {
		response = append(response, models.VideoResponse{
			ID:           v.ID,
			Title:        v.Title,
			Status:       v.Status,
			Duration:     *v.Duration,
			ThumbnailURL: h.storage.GetObjectURL(*v.ThumbnailKey),
			StreamURL:    h.storage.GetObjectURL(v.VideoKey),
		})
	}
	return c.JSON(response)
}

// GET /api/v1/videos/:id/stream - Stream video
func (h *Handlers) StreamVideo(c *fiber.Ctx) error {
	id := c.Params("id")
	video, err := h.videoSvc.GetVideoByID(c.Context(), id)
	if err != nil || video.Status != "ready" {
		return c.Status(404).JSON(fiber.Map{"error": "Video not ready"})
	}

	// Redirect to MinIO presigned URL (60min expiry)
	url, err := h.storage.GetPresignedURL(c.Context(), video.VideoKey, 60*time.Minute)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to generate stream URL"})
	}

	return c.Redirect(url, 302)
}

// Background processing
func (h *Handlers) processVideoAsync(ctx context.Context, videoID uuid.UUID) {
	log.Printf("Processing video %s", videoID)

	video, err := h.videoSvc.GetVideoByID(ctx, videoID.String())
	if err != nil {
		log.Printf("Failed to get video %s: %v", videoID, err)
		return
	}

	// Generate thumbnail at 2 seconds
	thumbnailFilename := fmt.Sprintf("%s.jpg", videoID)
	defer os.Remove(thumbnailFilename)
	if addInfo, err := h.generateAddInfo(ctx, video.VideoKey, thumbnailFilename); err != nil {
		log.Printf("Thumbnail failed for %s: %v", videoID, err)
	} else {
		thumbnailKey, _ := h.storage.SaveFileWithReader(ctx, thumbnailFilename, "image/jpeg", addInfo.pr) // Load actual image bytes
		h.videoSvc.UpdateAddInfo(ctx, videoID, thumbnailKey, addInfo.duration)
	}

	// Update status
	h.videoSvc.UpdateStatus(ctx, videoID, "ready")
	log.Printf("Video %s processing complete", videoID)
}

func (h *Handlers) generateAddInfo(ctx context.Context, videoKey, outputName string) (AddInfoResponse, error) {
	pr, pw := io.Pipe()

	url, _ := h.storage.GetPresignedURL(ctx, videoKey, time.Duration(15)*time.Minute)
	// FFmpeg command: extract frame at 2 seconds
	cmd := exec.Command("ffmpeg",
		"-ss", "00:00:02", // Seek to 2nd second (fast)
		"-i", url, // Input from MinIO
		"-vframes", "1", // 1 frame only
		"-f", "image2pipe", // Tell FFmpeg to treat output as a stream
		"-vcodec", "mjpeg",
		"pipe:1",
	)
	cmd.Stdout = pw
	go func() {
		err := cmd.Run()
		pw.CloseWithError(err)
	}()

	duration, _ := h.GetVideoDuration(url)
	return AddInfoResponse{duration: duration, pr: pr}, nil
}

func (h *Handlers) GetVideoDuration(videoURL string) (int, error) {
	// ffprobe command
	// -v error: hide logs except errors
	// -show_entries format=duration: only get the duration
	// -of default=noprint_wrappers=1:nocopyright=1: simplify the output
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		videoURL,
	)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return 0, fmt.Errorf("ffprobe error: %v", err)
	}

	// ffprobe returns duration as a string like "120.450000"
	durationStr := strings.TrimSpace(out.String())
	durationFloat, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid duration format: %v", err)
	}

	// Convert to seconds (integer)
	return int(durationFloat), nil
}
