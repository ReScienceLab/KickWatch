package config

import "os"

type Config struct {
	DatabaseURL  string
	Port         string
	APNSKeyID    string
	APNSTeamID   string
	APNSBundleID string
	APNSKeyPath  string
	APNSKey      string
	APNSEnv      string
	ProxyURL     string
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
	return &Config{
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		Port:         port,
		APNSKeyID:    os.Getenv("APNS_KEY_ID"),
		APNSTeamID:   os.Getenv("APNS_TEAM_ID"),
		APNSBundleID: os.Getenv("APNS_BUNDLE_ID"),
		APNSKeyPath:  os.Getenv("APNS_KEY_PATH"),
		APNSKey:      os.Getenv("APNS_KEY"),
		APNSEnv:      apnsEnv,
		ProxyURL:     os.Getenv("WEBSHARE_PROXY_URL"),
	}
}
