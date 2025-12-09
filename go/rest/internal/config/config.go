package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port     uint
	LogLevel string
}

func New() (*Config, error) {
	port, err := getDefaultUintEnv("PORT", 8080)
	if err != nil {
		return nil, fmt.Errorf("failed to get PORT: %w", err)
	}

	logLevel := getDefaultStringEnv("LOG_LEVEL", "INFO")

	return &Config{
		Port:     port,
		LogLevel: logLevel,
	}, nil
}

func getDefaultStringEnv(key string, defaultVal string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVal
}

func getDefaultUintEnv(key string, defaultValue uint) (uint, error) {
	v := os.Getenv(key)
	if len(v) == 0 {
		return defaultValue, nil
	}

	ret, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid environment variable %s=%s: %w", key, v, err)
	}
	return uint(ret), nil
}
