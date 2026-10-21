package config

import (
	"os"
)

// Config holds runtime configuration values.
type Config struct {
	Port          string
	DBPath        string
	JWTSecret     string
	PublicBaseURL string
}

// LoadConfig loads environment variables with sane defaults.
func LoadConfig() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/classroom.db"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "classroom-dev-secret-key-change-in-production-2026"
	}

	publicBaseURL := os.Getenv("PUBLIC_BASE_URL")
	if publicBaseURL == "" {
		publicBaseURL = "http://localhost:" + port
	}

	return &Config{
		Port:          port,
		DBPath:        dbPath,
		JWTSecret:     jwtSecret,
		PublicBaseURL: publicBaseURL,
	}
}
