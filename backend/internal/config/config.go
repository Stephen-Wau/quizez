package config

import (
	"os"
	"strconv"
)

type Config struct {
	DBUser          string
	DBPassword      string
	DBHost          string
	DBPort          string
	DBName          string
	FrontendOrigin  string
	JWTSecret       string
	JWTExpiryHours  int
}

// Load baca konfigurasi dari environment variable (biasanya diisi lewat .env), tiap key ada
// fallback default-nya biar app tetap bisa jalan pas development walau .env belum lengkap.
func Load() Config {
	return Config{
		DBUser:         getEnv("DB_USER", "root"),
		DBPassword:     getEnv("DB_PASSWORD", ""),
		DBHost:         getEnv("DB_HOST", "127.0.0.1"),
		DBPort:         getEnv("DB_PORT", "3306"),
		DBName:         getEnv("DB_NAME", "quizez_db"),
		FrontendOrigin: getEnv("FRONTEND_ORIGIN", "http://localhost:4200"),
		JWTSecret:      getEnv("JWT_SECRET", "change-me-to-a-long-random-string"),
		JWTExpiryHours: getEnvInt("JWT_EXPIRY_HOURS", 24),
	}
}

// getEnv ambil env var string, pakai fallback kalau belum diset (kosong).
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getEnvInt sama kayak getEnv tapi buat nilai angka, fallback juga dipakai kalau env-nya gak valid int.
func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
