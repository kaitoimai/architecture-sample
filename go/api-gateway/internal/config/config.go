package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config はAPI Gatewayの設定全体
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Logging LoggingConfig `yaml:"logging"`
	Routing RoutingConfig `yaml:"routing"`
	Redis   RedisConfig   `yaml:"redis,omitempty"`
	JWT     JWTConfig     `yaml:"jwt,omitempty"`
}

// ServerConfig はHTTPサーバの設定
type ServerConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

// LoggingConfig はログの設定
type LoggingConfig struct {
	Level  string `yaml:"level"`  // debug, info, warn, error
	Format string `yaml:"format"` // json, text
}

// RoutingConfig はルーティングの設定
type RoutingConfig struct {
	ConfigFile      string `yaml:"config_file"`
	EnableHotReload bool   `yaml:"enable_hot_reload"`
}

// RedisConfig はRedisの設定
type RedisConfig struct {
	Host         string        `yaml:"host"`
	Password     string        `yaml:"password"`
	DB           int           `yaml:"db"`
	PoolSize     int           `yaml:"pool_size"`
	DialTimeout  time.Duration `yaml:"dial_timeout"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	KeyPrefix    string        `yaml:"key_prefix"` // Revoke情報のキープレフィックス
}

// JWTConfig はJWT検証の設定
type JWTConfig struct {
	// PublicKeyFiles は公開鍵ファイルのパス (kid → ファイルパス)
	PublicKeyFiles map[string]string `yaml:"public_key_files,omitempty"`
	// SkipValidation は検証をスキップするか（開発環境用）
	SkipValidation bool `yaml:"skip_validation,omitempty"`
}

// Route はルーティング設定の1つのルート
type Route struct {
	Path       string             `yaml:"path"`
	Methods    []string           `yaml:"methods"`
	Backend    BackendConfig      `yaml:"backend"`
	Middleware []MiddlewareConfig `yaml:"middleware,omitempty"`
	Priority   int                `yaml:"priority"`
}

// BackendConfig はバックエンドの設定
type BackendConfig struct {
	URL     string        `yaml:"url"`
	Timeout time.Duration `yaml:"timeout"`
}

// MiddlewareConfig はミドルウェアの設定
type MiddlewareConfig struct {
	Type   string         `yaml:"type"`
	Config map[string]any `yaml:"config,omitempty"`
}

// RoutingFileConfig はルーティング設定ファイルの構造
type RoutingFileConfig struct {
	Routes []Route `yaml:"routes"`
}

// LoadConfig は設定ファイルを読み込む
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// LoadRoutingConfig はルーティング設定ファイルを読み込む
func LoadRoutingConfig(path string) (*RoutingFileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read routing config file: %w", err)
	}

	var cfg RoutingFileConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal routing config: %w", err)
	}

	return &cfg, nil
}

// Validate は設定の妥当性を検証する
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Server.ReadTimeout <= 0 {
		return fmt.Errorf("read_timeout must be positive")
	}

	if c.Server.WriteTimeout <= 0 {
		return fmt.Errorf("write_timeout must be positive")
	}

	if c.Routing.ConfigFile == "" {
		return fmt.Errorf("routing config_file is required")
	}

	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", c.Logging.Level)
	}

	validFormats := map[string]bool{"json": true, "text": true}
	if !validFormats[c.Logging.Format] {
		return fmt.Errorf("invalid log format: %s", c.Logging.Format)
	}

	// Redis設定のバリデーション（オプション）
	if c.Redis.Host != "" {
		if c.Redis.DB < 0 {
			return fmt.Errorf("redis db must be non-negative")
		}
		if c.Redis.PoolSize < 0 {
			return fmt.Errorf("redis pool_size must be non-negative")
		}
		if c.Redis.DialTimeout < 0 {
			return fmt.Errorf("redis dial_timeout must be non-negative")
		}
		if c.Redis.ReadTimeout < 0 {
			return fmt.Errorf("redis read_timeout must be non-negative")
		}
		if c.Redis.WriteTimeout < 0 {
			return fmt.Errorf("redis write_timeout must be non-negative")
		}
	}

	return nil
}

// Address はサーバのアドレスを返す
func (s *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}
