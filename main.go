package main

import (
	"log"

	"github.com/joho/godotenv"
	"tka-learning-portal/file-service/config"
	"tka-learning-portal/file-service/database"
	"tka-learning-portal/file-service/handler"
	"tka-learning-portal/file-service/repository"
	"tka-learning-portal/file-service/router"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	if cfg.JWTPublicKey == nil {
		log.Fatal("JWT_PUBLIC_KEY is required")
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("cannot connect to database: %v", err)
	}
	defer db.Close()

	fileRepo := repository.NewFileRepository(db)
	fileHandler := handler.NewFileHandler(fileRepo, cfg.UploadDir, cfg.MaxFileSizeMB)

	r := router.Setup(fileHandler, cfg.JWTPublicKey)

	log.Printf("file service starting on :%s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
