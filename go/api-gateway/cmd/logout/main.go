package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"api-gateway/internal/config"
	"api-gateway/internal/handler"
	"api-gateway/internal/repository"
	"api-gateway/pkg/logger"
	redisclient "api-gateway/pkg/redis"
)

func main() {
	// コマンドライン引数のパース
	configPath := flag.String("config", "configs/gateway.yaml", "path to config file")
	flag.Parse()

	// 設定ファイルの読み込み
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// ロガーの初期化
	log := logger.New(logger.Config{
		Level:  logger.LogLevel(cfg.Logging.Level),
		Format: cfg.Logging.Format,
	})

	log.Info("starting logout server",
		"version", "1.0.0",
		"config", *configPath)

	// Redis設定の確認
	if cfg.Redis.Host == "" {
		log.Error("redis host is not configured")
		os.Exit(1)
	}

	// Redisクライアントの初期化
	redisClient, err := redisclient.NewClient(redisclient.Config{
		Host:         cfg.Redis.Host,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
	})
	if err != nil {
		log.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	log.Info("redis connected successfully", "host", cfg.Redis.Host, "db", cfg.Redis.DB)

	// セッションリポジトリの初期化
	sessionRepo := repository.NewRedisSessionRepository(redisClient, cfg.Redis.KeyPrefix)

	// Logoutハンドラの初期化
	logoutHandler := handler.NewLogoutHandler(handler.LogoutConfig{
		Repository:    sessionRepo,
		UserIDClaim:   "sub", // 設定可能にする場合は cfg に追加
		JWTExpiration: 10 * time.Hour,
		Logger:        log,
	})

	// HTTPサーバーの設定
	server := &http.Server{
		Addr:         cfg.Server.Address(),
		Handler:      logoutHandler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// グレースフルシャットダウンの設定
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// サーバー起動
	go func() {
		log.Info("logout server is running", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// シャットダウンシグナル待機
	<-quit
	log.Info("shutting down logout server...")

	// グレースフルシャットダウン
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	log.Info("logout server stopped")
}
