package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/raulviigipuu/deploy_ftp/internal/logx"
)

type UploadEntry struct {
	Local  string `json:"local"`
	Remote string `json:"remote"`
}

type Config struct {
	FTPHost string
	FTPPort int
	FTPUser string
	FTPPass string
	UseTLS  bool
	Map     []UploadEntry
}

// ===========================
// Load Config
// ===========================

func Load(envPath string, mapPath string) (*Config, error) {
	cfg := &Config{}

	// 1. Load .env
	if envPath == "" {
		envPath = filepath.Join(".", ".env")
	}
	if err := godotenv.Load(envPath); err != nil {
		logx.Info(fmt.Sprintf("⚠️ Could not load .env from %s: %v", envPath, err))
	}

	// Extract and validate required credentials
	if err := validateEnv([]string{"FTP_HOST", "FTP_USER", "FTP_PASS", "FTP_PORT"}); err != nil {
		return nil, err
	}

	cfg.FTPHost = os.Getenv("FTP_HOST")
	cfg.FTPUser = os.Getenv("FTP_USER")
	cfg.FTPPass = os.Getenv("FTP_PASS")

	// Validate and parse FTP_PORT
	portStr := os.Getenv("FTP_PORT")
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return nil, fmt.Errorf("❌ Invalid FTP_PORT: must be a number between 1 and 65535")
	}
	cfg.FTPPort = port

	// Optional: use TLS if FTP_TLS=true
	cfg.UseTLS = strings.ToLower(os.Getenv("FTP_TLS")) == "true"

	// 2. Load upload_map.json
	if mapPath == "" {
		mapPath = filepath.Join(".", "upload_map.json")
	}
	mapData, err := os.ReadFile(mapPath)
	if err != nil {
		return nil, fmt.Errorf("❌ Could not read upload_map.json: %v\nHint: Provide it via --map or place it in current dir", err)
	}

	if err := json.Unmarshal(mapData, &cfg.Map); err != nil {
		return nil, fmt.Errorf("❌ Invalid JSON in upload_map.json: %v", err)
	}

	return cfg, nil
}

// ===========================
// Helpers
// ===========================

func validateEnv(keys []string) error {
	for _, key := range keys {
		if os.Getenv(key) == "" {
			return fmt.Errorf("❌ Missing required environment variable: %s", key)
		}
	}
	return nil
}
