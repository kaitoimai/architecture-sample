package main

import (
	"os"

	"github.com/kaitoimai/go-sample/rest/internal/config"
	"github.com/kaitoimai/go-sample/rest/internal/pkg/logger"
	"github.com/kaitoimai/go-sample/rest/internal/server"
)

func main() {
	log := logger.NewFromEnv()
	logger.SetDefault(log)

	cfg, err := config.New()
	if err != nil {
		log.Error("failed to initialize application", "err", err)
		os.Exit(1)
	}

	srv, err := server.New(cfg, log)
	if err != nil {
		log.Error("failed to create server", "err", err)
		os.Exit(1)
	}
	if err := srv.Start(); err != nil {
		log.Error("server error", "err", err)
		os.Exit(1)
	}
}
