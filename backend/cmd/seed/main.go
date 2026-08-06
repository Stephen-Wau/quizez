package main

import (
	"log"

	"github.com/joho/godotenv"

	"quizez/backend/internal/auth"
	"quizez/backend/internal/config"
	"quizez/backend/internal/db"
)

const (
	adminEmail    = "admin@mail.com"
	adminPassword = "password123"
	adminName     = "Admin"
)

// main seed/reset akun admin default buat kebutuhan development & testing lokal. Pakai
// ON DUPLICATE KEY UPDATE jadi aman dijalanin berkali-kali (idempotent), gak bikin duplikat.
func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	conn, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer conn.Close()

	hash, err := auth.HashPassword(adminPassword)
	if err != nil {
		log.Fatalf("failed to hash password: %v", err)
	}

	_, err = conn.Exec(`
		INSERT INTO users (email, password_hash, name)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE password_hash = VALUES(password_hash), name = VALUES(name)
	`, adminEmail, hash, adminName)
	if err != nil {
		log.Fatalf("failed to seed admin user: %v", err)
	}

	log.Printf("seeded admin user: %s / %s\n", adminEmail, adminPassword)
}
