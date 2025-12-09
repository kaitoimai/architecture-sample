package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		content   string
		wantErr   bool
		validate  func(*testing.T, *Config)
	}{
		{
			name: "valid config",
			content: `
server:
  host: "127.0.0.1"
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  shutdown_timeout: 10s

logging:
  level: "info"
  format: "json"

routing:
  config_file: "routes.yaml"
  enable_hot_reload: false
`,
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Server.Host != "127.0.0.1" {
					t.Errorf("Host = %s, want 127.0.0.1", cfg.Server.Host)
				}
				if cfg.Server.Port != 8080 {
					t.Errorf("Port = %d, want 8080", cfg.Server.Port)
				}
				if cfg.Logging.Level != "info" {
					t.Errorf("Level = %s, want info", cfg.Logging.Level)
				}
			},
		},
		{
			name: "invalid port",
			content: `
server:
  host: "0.0.0.0"
  port: 99999
  read_timeout: 30s
  write_timeout: 30s
  shutdown_timeout: 10s

logging:
  level: "info"
  format: "json"

routing:
  config_file: "routes.yaml"
`,
			wantErr: true,
		},
		{
			name: "invalid log level",
			content: `
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  shutdown_timeout: 10s

logging:
  level: "invalid"
  format: "json"

routing:
  config_file: "routes.yaml"
`,
			wantErr: true,
		},
		{
			name: "missing routing config file",
			content: `
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  shutdown_timeout: 10s

logging:
  level: "info"
  format: "json"

routing:
  config_file: ""
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テスト用の設定ファイルを作成
			configPath := filepath.Join(tempDir, tt.name+".yaml")
			if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			cfg, err := LoadConfig(configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("LoadConfig() expected error for nonexistent file, got nil")
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Server: ServerConfig{
					Port:            8080,
					ReadTimeout:     30 * time.Second,
					WriteTimeout:    30 * time.Second,
					ShutdownTimeout: 10 * time.Second,
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
				},
				Routing: RoutingConfig{
					ConfigFile: "routes.yaml",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid port - negative",
			config: Config{
				Server: ServerConfig{
					Port:         -1,
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
				},
				Routing: RoutingConfig{
					ConfigFile: "routes.yaml",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid port - too large",
			config: Config{
				Server: ServerConfig{
					Port:         99999,
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
				},
				Routing: RoutingConfig{
					ConfigFile: "routes.yaml",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid read timeout",
			config: Config{
				Server: ServerConfig{
					Port:         8080,
					ReadTimeout:  0,
					WriteTimeout: 30 * time.Second,
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
				},
				Routing: RoutingConfig{
					ConfigFile: "routes.yaml",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid write timeout",
			config: Config{
				Server: ServerConfig{
					Port:         8080,
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 0,
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
				},
				Routing: RoutingConfig{
					ConfigFile: "routes.yaml",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			config: Config{
				Server: ServerConfig{
					Port:         8080,
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
				},
				Logging: LoggingConfig{
					Level:  "invalid",
					Format: "json",
				},
				Routing: RoutingConfig{
					ConfigFile: "routes.yaml",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid log format",
			config: Config{
				Server: ServerConfig{
					Port:         8080,
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "invalid",
				},
				Routing: RoutingConfig{
					ConfigFile: "routes.yaml",
				},
			},
			wantErr: true,
		},
		{
			name: "missing routing config file",
			config: Config{
				Server: ServerConfig{
					Port:         8080,
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
				},
				Routing: RoutingConfig{
					ConfigFile: "",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerConfigAddress(t *testing.T) {
	tests := []struct {
		name   string
		config ServerConfig
		want   string
	}{
		{
			name: "default host and port",
			config: ServerConfig{
				Host: "0.0.0.0",
				Port: 8080,
			},
			want: "0.0.0.0:8080",
		},
		{
			name: "localhost",
			config: ServerConfig{
				Host: "localhost",
				Port: 3000,
			},
			want: "localhost:3000",
		},
		{
			name: "empty host",
			config: ServerConfig{
				Host: "",
				Port: 8080,
			},
			want: ":8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.Address()
			if got != tt.want {
				t.Errorf("Address() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestLoadRoutingConfig(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		content  string
		wantErr  bool
		validate func(*testing.T, *RoutingFileConfig)
	}{
		{
			name: "valid routing config",
			content: `
routes:
  - path: "/api/v1/users"
    methods: ["GET", "POST"]
    backend:
      url: "https://user-service.example.com"
      timeout: 30s
    middleware:
      - type: "jwt"
    priority: 10
`,
			wantErr: false,
			validate: func(t *testing.T, cfg *RoutingFileConfig) {
				if len(cfg.Routes) != 1 {
					t.Fatalf("Routes length = %d, want 1", len(cfg.Routes))
				}
				route := cfg.Routes[0]
				if route.Path != "/api/v1/users" {
					t.Errorf("Path = %s, want /api/v1/users", route.Path)
				}
				if len(route.Methods) != 2 {
					t.Errorf("Methods length = %d, want 2", len(route.Methods))
				}
				if route.Backend.URL != "https://user-service.example.com" {
					t.Errorf("Backend URL = %s, want https://user-service.example.com", route.Backend.URL)
				}
			},
		},
		{
			name:    "empty file",
			content: ``,
			wantErr: false,
			validate: func(t *testing.T, cfg *RoutingFileConfig) {
				if len(cfg.Routes) != 0 {
					t.Errorf("Routes length = %d, want 0", len(cfg.Routes))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			routingPath := filepath.Join(tempDir, tt.name+".yaml")
			if err := os.WriteFile(routingPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test routing config: %v", err)
			}

			cfg, err := LoadRoutingConfig(routingPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadRoutingConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

func TestLoadRoutingConfig_FileNotFound(t *testing.T) {
	_, err := LoadRoutingConfig("/nonexistent/path/routing.yaml")
	if err == nil {
		t.Error("LoadRoutingConfig() expected error for nonexistent file, got nil")
	}
}
