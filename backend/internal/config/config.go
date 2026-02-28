package config

import (
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL  string
	Port         string
	APNSKeyID    string
	APNSTeamID   string
	APNSBundleID string
	APNSKeyPath  string
	APNSKey      string
	APNSEnv      string
	// ScrapingBee configuration
	ScrapingBeeAPIKey        string
	ScrapingBeeMaxConcurrent int
	// Vertex AI configuration
	VertexAIProjectID string
	VertexAILocation  string
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	apnsEnv := os.Getenv("APNS_ENV")
	if apnsEnv == "" {
		apnsEnv = "sandbox"
	}

	maxConcurrent := 10 // default
	if val := os.Getenv("SCRAPINGBEE_MAX_CONCURRENT"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			maxConcurrent = parsed
		}
	}

	return &Config{
		DatabaseURL:              os.Getenv("DATABASE_URL"),
		Port:                     port,
		APNSKeyID:                os.Getenv("APNS_KEY_ID"),
		APNSTeamID:               os.Getenv("APNS_TEAM_ID"),
		APNSBundleID:             os.Getenv("APNS_BUNDLE_ID"),
		APNSKeyPath:              os.Getenv("APNS_KEY_PATH"),
		APNSKey:                  os.Getenv("APNS_KEY"),
		APNSEnv:                  apnsEnv,
		ScrapingBeeAPIKey:        os.Getenv("SCRAPINGBEE_API_KEY"),
		ScrapingBeeMaxConcurrent: maxConcurrent,
		VertexAIProjectID:        os.Getenv("VERTEX_AI_PROJECT_ID"),
		VertexAILocation:         os.Getenv("VERTEX_AI_LOCATION"),
	}
}
