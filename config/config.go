package config

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DatabaseURL  string
	JWTPublicKey *rsa.PublicKey
	ServerPort   string
	UploadDir    string
	MaxFileSizeMB int64
}

func Load() *Config {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:root@localhost:5432/learning_portal_v2?sslmode=disable"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}

	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}

	maxSizeMB := int64(10)
	if v := os.Getenv("MAX_FILE_SIZE_MB"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			maxSizeMB = n
		}
	}

	cfg := &Config{
		DatabaseURL:   dbURL,
		ServerPort:    port,
		UploadDir:     uploadDir,
		MaxFileSizeMB: maxSizeMB,
	}

	if keyPEM := os.Getenv("JWT_PUBLIC_KEY"); keyPEM != "" {
		cfg.JWTPublicKey = parsePublicKey(keyPEM)
	}

	return cfg
}

func parsePublicKey(pemStr string) *rsa.PublicKey {
	pemStr = strings.ReplaceAll(pemStr, `\n`, "\n")
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		log.Fatal("JWT_PUBLIC_KEY: failed to decode PEM block")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		log.Fatalf("JWT_PUBLIC_KEY: failed to parse key: %v", err)
	}
	key, ok := parsed.(*rsa.PublicKey)
	if !ok {
		log.Fatal("JWT_PUBLIC_KEY: not an RSA public key")
	}
	return key
}
