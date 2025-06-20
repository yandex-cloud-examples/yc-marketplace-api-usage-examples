package config

import (
	"fmt"
	"log/slog"
	"os"
)

// Config holds the application configuration.
type Config struct {
	Port                  string
	YDBConnectionString   string
	ServiceAccountKeyFile string
	DefaultSkuID          string
}

// LoadConfig loads configuration from environment variables and sets defaults.
func LoadConfig(logger *slog.Logger) (*Config, error) {
	cfg := &Config{
		Port:                  os.Getenv("PORT"),
		YDBConnectionString:   os.Getenv("YDB_CONNECTION_STRING"),
		ServiceAccountKeyFile: os.Getenv("YC_SA_KEY_FILE"),
		DefaultSkuID:          os.Getenv("DEFAULT_SKU_ID"),
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
		logger.Info("PORT not set, defaulting to 8080")
	}

	if cfg.YDBConnectionString == "" {
		return nil, fmt.Errorf("YDB_CONNECTION_STRING environment variable is required but not set")
	}

	if cfg.DefaultSkuID == "" {
		// Using the previous hardcoded demo SKU ID as a fallback if not set via env var.
		// Ideally, this should also be an error or a well-defined default from a constants file.
		cfg.DefaultSkuID = "dn2e395635hoblkrj1at"
		logger.Warn("DEFAULT_SKU_ID not set, using default demo SKU ID", slog.String("default_sku_id", cfg.DefaultSkuID))
	}

	logger.Info("Configuration loaded",
		slog.String("port", cfg.Port),
		slog.Bool("sa_key_file_set", cfg.ServiceAccountKeyFile != ""),
		slog.String("default_sku_id", cfg.DefaultSkuID),
	)
	return cfg, nil
}
