package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	ogenmw "github.com/ogen-go/ogen/middleware"

	"github.com/kaitoimai/go-sample/rest/internal/config"
	"github.com/kaitoimai/go-sample/rest/internal/handler"
	"github.com/kaitoimai/go-sample/rest/internal/middleware"
	"github.com/kaitoimai/go-sample/rest/internal/oas"
	logx "github.com/kaitoimai/go-sample/rest/internal/pkg/logger"
)

const (
	gracefulShutdownTimeout = 10 * time.Second

	// NOTE: 以下のタイムアウト値はベースライン。運用要件に応じて調整する。
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 10 * time.Second
	idleTimeout       = 60 * time.Second
)

type Server struct {
	httpServer *http.Server
	config     *config.Config
	logger     *slog.Logger
}

func New(cfg *config.Config, logger *slog.Logger) (*Server, error) {
	// Create middlewares
	authnMiddleware := middleware.NewAuthnMiddleware()
	authzMiddleware := middleware.NewAuthzMiddleware()

	// Create OAS handler
	oasHandler := handler.NewOASHandler()

	// Create OAS server
	oasServer, err := oas.NewServer(
		oasHandler,
		oas.WithMiddleware(func(req ogenmw.Request, next ogenmw.Next) (ogenmw.Response, error) {
			// リクエスト固有の情報（method/path）をログに自動付与するため、request-scoped loggerを作成してContextに保存
			reqLogger := logger.With("method", req.Raw.Method, "path", req.Raw.URL.Path)
			req.Context = logx.NewContext(req.Context, reqLogger)
			logger.Info("request", "method", req.Raw.Method, "path", req.Raw.URL.Path)
			return next(req)
		}),
		oas.WithMiddleware(authnMiddleware.Handle), // API Gateway検証済みJWTからClaims抽出
		oas.WithMiddleware(authzMiddleware.Handle), // RBAC認可（ロールベースアクセス制御）
		oas.WithErrorHandler(middleware.ErrorHandler),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OAS server: %w", err)
	}

	return &Server{
		httpServer: &http.Server{
			Addr:              fmt.Sprintf(":%d", cfg.Port),
			Handler:           oasServer,
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
		},
		config: cfg,
		logger: logger,
	}, nil
}

func (s *Server) Start() error {
	serverCtx, serverStopCtx := context.WithCancel(context.Background())
	timeoutCh := make(chan struct{}, 1)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sig)

	go func() {
		<-sig

		shutdownCtx, cancel := context.WithTimeout(serverCtx, gracefulShutdownTimeout)
		defer cancel()

		s.logger.Info("gracefully shutting down...")
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("error during server shutdown", "err", err)
		}

		if shutdownCtx.Err() == context.DeadlineExceeded {
			s.logger.Warn("graceful shutdown timed out")
			select {
			case timeoutCh <- struct{}{}:
			default:
			}
		}

		serverStopCtx()
	}()

	s.logger.Info("server is running", "port", s.config.Port)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("error starting server: %w", err)
	}

	<-serverCtx.Done()

	select {
	case <-timeoutCh:
		return fmt.Errorf("graceful shutdown timed out")
	default:
	}

	s.logger.Info("server gracefully shutdown")
	return nil
}
