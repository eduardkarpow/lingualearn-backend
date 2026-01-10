package main

import (
	"database/sql"
	"fmt"
	"linguaLearn/internal/handlers"
	"linguaLearn/internal/services"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	err := godotenv.Load("/app/.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	fmt.Printf("DSN received: [%s]\n", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Println("failed to open video")
	}
	err = db.Ping()
	if err != nil {
		log.Fatalf("error pinging db: %s", err)
	}

	storage, _ := services.NewStorageService(
		"minio:9000", "minioadmin", "minioadmin", "polylearn",
	)

	videoSvc := services.NewVideoService(db)
	handlers := handlers.NewHandlers(videoSvc, storage)

	app := fiber.New(fiber.Config{
		BodyLimit:    32 << 20,
		Prefork:      false,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		AppName:      "PolyLearn",
	})

	// Middlewares
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{AllowOrigins: "*"}))

	// Routes
	api := app.Group("/api/v1")
	api.Post("/video", handlers.UploadVideo)
	api.Get("/test", handlers.TestFiber)
	api.Get("/videos", handlers.ListVideos)
	api.Get("/videos/:id/stream", handlers.StreamVideo)
	api.Post("/sub", handlers.UploadSubtitles)
	api.Get("/sub/:videoId", handlers.GetSubs)
	api.Put("/sub/shift", handlers.ShiftSubs)
	//api.Get("/videos/:id/thumbnail", handlers.GetThumbnail)

	log.Fatal(app.Listen(":8000"))
}
