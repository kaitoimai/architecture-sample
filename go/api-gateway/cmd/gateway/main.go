package main

import (
	"context"
	"crypto/rsa"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"api-gateway/internal/config"
	"api-gateway/internal/handler"
	"api-gateway/internal/middleware"
	"api-gateway/internal/middleware/auth"
	"api-gateway/internal/repository"
	"api-gateway/internal/routing"
	"api-gateway/internal/transport"
	"api-gateway/pkg/logger"
	"api-gateway/pkg/redis"
)

func main() {
	// コマンドライン引数のパース
	configPath := flag.String("config", "configs/gateway.yaml", "path to config file")
	flag.Parse()

	// 設定ファイルの読み込み
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// ロガーの初期化
	log := logger.New(logger.Config{
		Level:  logger.LogLevel(cfg.Logging.Level),
		Format: cfg.Logging.Format,
	})

	log.Info("Starting API Gateway",
		slog.String("version", "0.1.0"),
		slog.String("host", cfg.Server.Host),
		slog.Int("port", cfg.Server.Port),
	)

	// ルーティング設定の読み込み
	routingCfg, err := config.LoadRoutingConfig(cfg.Routing.ConfigFile)
	if err != nil {
		log.Error("Failed to load routing config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// ルーターの初期化
	router := routing.NewRouter()
	if err := router.LoadFromConfig(routingCfg); err != nil {
		log.Error("Failed to load routes", slog.String("error", err.Error()))
		os.Exit(1)
	}

	routes := router.GetAllRoutes()
	log.Info("Routes loaded", slog.Int("count", len(routes)))

	// Redisクライアントの初期化（設定がある場合）
	var sessionRepo repository.SessionRepository
	if cfg.Redis.Host != "" {
		redisClient, err := redis.NewClient(redis.Config{
			Host:         cfg.Redis.Host,
			Password:     cfg.Redis.Password,
			DB:           cfg.Redis.DB,
			PoolSize:     cfg.Redis.PoolSize,
			DialTimeout:  cfg.Redis.DialTimeout,
			ReadTimeout:  cfg.Redis.ReadTimeout,
			WriteTimeout: cfg.Redis.WriteTimeout,
		})
		if err != nil {
			log.Error("Failed to initialize Redis client", slog.String("error", err.Error()))
			os.Exit(1)
		}

		// Redis接続確認
		if err := redisClient.Ping(context.Background()); err != nil {
			log.Warn("Redis ping failed", slog.String("error", err.Error()))
		} else {
			log.Info("Redis connected successfully")
		}

		// セッションリポジトリの初期化
		sessionRepo = repository.NewRedisSessionRepository(redisClient, cfg.Redis.KeyPrefix)
	}

	// JWT公開鍵の読み込み（設定がある場合）
	var jwtPublicKeys map[string]*rsa.PublicKey
	if len(cfg.JWT.PublicKeyFiles) > 0 {
		keys, err := auth.LoadPublicKeysFromFiles(cfg.JWT.PublicKeyFiles)
		if err != nil {
			log.Error("Failed to load JWT public keys", slog.String("error", err.Error()))
			os.Exit(1)
		}
		jwtPublicKeys = keys
		log.Info("JWT public keys loaded", slog.Int("count", len(keys)))
	}

	// ミドルウェアファクトリーの初期化
	middlewareFactory := middleware.NewFactory(middleware.FactoryConfig{
		JWTPublicKeys: jwtPublicKeys,
		SessionRepo:   sessionRepo,
		Logger:        log,
	})

	// トランスポーターの初期化
	transporter := transport.NewHTTPTransporter()

	// Gatewayハンドラの初期化
	gateway := handler.NewGateway(router, transporter, middlewareFactory, log)

	// HTTPサーバの設定
	server := &http.Server{
		Addr:         cfg.Server.Address(),
		Handler:      gateway,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// サーバの起動
	go func() {
		log.Info("Server starting", slog.String("address", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// グレースフルシャットダウンの設定
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", slog.String("error", err.Error()))
		os.Exit(1)
	}

	log.Info("Server exited")
}
