package middleware

import (
	"crypto/rsa"
	"fmt"
	"log/slog"

	"api-gateway/internal/config"
	"api-gateway/internal/middleware/auth"
	"api-gateway/internal/repository"
)

// Factory はミドルウェアを生成するファクトリー
type Factory struct {
	jwtPublicKeys map[string]*rsa.PublicKey
	sessionRepo   repository.SessionRepository
	logger        *slog.Logger
}

// FactoryConfig はファクトリーの設定
type FactoryConfig struct {
	JWTPublicKeys map[string]*rsa.PublicKey
	SessionRepo   repository.SessionRepository
	Logger        *slog.Logger
}

// NewFactory は新しいファクトリーを作成する
func NewFactory(cfg FactoryConfig) *Factory {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return &Factory{
		jwtPublicKeys: cfg.JWTPublicKeys,
		sessionRepo:   cfg.SessionRepo,
		logger:        cfg.Logger,
	}
}

// Create は設定からミドルウェアを生成する
func (f *Factory) Create(cfg config.MiddlewareConfig) (Middleware, error) {
	switch cfg.Type {
	case "jwt":
		return f.createJWTMiddleware(cfg.Config)
	case "revoke":
		return f.createRevokeMiddleware(cfg.Config)
	case "cors":
		return f.createCORSMiddleware(cfg.Config)
	case "logging":
		return f.createLoggingMiddleware(cfg.Config)
	case "recovery":
		return f.createRecoveryMiddleware(cfg.Config)
	default:
		return nil, fmt.Errorf("unknown middleware type: %s", cfg.Type)
	}
}

// createJWTMiddleware はJWT認証ミドルウェアを生成する
func (f *Factory) createJWTMiddleware(cfg map[string]any) (Middleware, error) {
	jwtConfig := auth.JWTConfig{
		PublicKeys:     f.jwtPublicKeys,
		SkipValidation: false,
		RequiredClaims: []string{},
	}

	// skip_validation の設定
	if skipVal, ok := cfg["skip_validation"]; ok {
		if skip, ok := skipVal.(bool); ok {
			jwtConfig.SkipValidation = skip
		}
	}

	// required_claims の設定
	if claimsVal, ok := cfg["required_claims"]; ok {
		if claims, ok := claimsVal.([]any); ok {
			for _, claim := range claims {
				if claimStr, ok := claim.(string); ok {
					jwtConfig.RequiredClaims = append(jwtConfig.RequiredClaims, claimStr)
				}
			}
		}
	}

	return auth.NewJWTMiddleware(jwtConfig), nil
}

// createRevokeMiddleware はRevoke検証ミドルウェアを生成する
func (f *Factory) createRevokeMiddleware(cfg map[string]any) (Middleware, error) {
	if f.sessionRepo == nil {
		return nil, fmt.Errorf("session repository is required for revoke middleware")
	}

	revokeConfig := auth.RevokeConfig{
		Repository:    f.sessionRepo,
		UserIDClaim:   "sub",
		IssuedAtClaim: "iat",
		FailOpen:      false,
		Logger:        f.logger,
	}

	// fail_open の設定
	if failOpenVal, ok := cfg["fail_open"]; ok {
		if failOpen, ok := failOpenVal.(bool); ok {
			revokeConfig.FailOpen = failOpen
		}
	}

	// user_id_claim の設定
	if userIDClaimVal, ok := cfg["user_id_claim"]; ok {
		if userIDClaim, ok := userIDClaimVal.(string); ok {
			revokeConfig.UserIDClaim = userIDClaim
		}
	}

	// issued_at_claim の設定
	if issuedAtClaimVal, ok := cfg["issued_at_claim"]; ok {
		if issuedAtClaim, ok := issuedAtClaimVal.(string); ok {
			revokeConfig.IssuedAtClaim = issuedAtClaim
		}
	}

	return auth.NewRevokeMiddleware(revokeConfig), nil
}

// createCORSMiddleware はCORSミドルウェアを生成する
func (f *Factory) createCORSMiddleware(cfg map[string]any) (Middleware, error) {
	corsConfig := CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		ExposedHeaders:   []string{},
		AllowCredentials: false,
		MaxAge:           3600,
	}

	// allowed_origins の設定
	if originsVal, ok := cfg["allowed_origins"]; ok {
		if origins, ok := originsVal.([]any); ok {
			corsConfig.AllowedOrigins = []string{}
			for _, origin := range origins {
				if originStr, ok := origin.(string); ok {
					corsConfig.AllowedOrigins = append(corsConfig.AllowedOrigins, originStr)
				}
			}
		}
	}

	// allowed_methods の設定
	if methodsVal, ok := cfg["allowed_methods"]; ok {
		if methods, ok := methodsVal.([]any); ok {
			corsConfig.AllowedMethods = []string{}
			for _, method := range methods {
				if methodStr, ok := method.(string); ok {
					corsConfig.AllowedMethods = append(corsConfig.AllowedMethods, methodStr)
				}
			}
		}
	}

	// allowed_headers の設定
	if headersVal, ok := cfg["allowed_headers"]; ok {
		if headers, ok := headersVal.([]any); ok {
			corsConfig.AllowedHeaders = []string{}
			for _, header := range headers {
				if headerStr, ok := header.(string); ok {
					corsConfig.AllowedHeaders = append(corsConfig.AllowedHeaders, headerStr)
				}
			}
		}
	}

	// exposed_headers の設定
	if exposedVal, ok := cfg["exposed_headers"]; ok {
		if exposed, ok := exposedVal.([]any); ok {
			for _, header := range exposed {
				if headerStr, ok := header.(string); ok {
					corsConfig.ExposedHeaders = append(corsConfig.ExposedHeaders, headerStr)
				}
			}
		}
	}

	// allow_credentials の設定
	if credsVal, ok := cfg["allow_credentials"]; ok {
		if creds, ok := credsVal.(bool); ok {
			corsConfig.AllowCredentials = creds
		}
	}

	// max_age の設定
	if maxAgeVal, ok := cfg["max_age"]; ok {
		if maxAge, ok := maxAgeVal.(int); ok {
			corsConfig.MaxAge = maxAge
		}
	}

	return NewCORSMiddleware(corsConfig), nil
}

// createLoggingMiddleware はログミドルウェアを生成する
func (f *Factory) createLoggingMiddleware(cfg map[string]any) (Middleware, error) {
	loggingConfig := LoggingConfig{
		LogRequestBody:  false,
		LogResponseBody: false,
		SkipPaths:       []string{},
	}

	// log_request_body の設定
	if logReqVal, ok := cfg["log_request_body"]; ok {
		if logReq, ok := logReqVal.(bool); ok {
			loggingConfig.LogRequestBody = logReq
		}
	}

	// log_response_body の設定
	if logResVal, ok := cfg["log_response_body"]; ok {
		if logRes, ok := logResVal.(bool); ok {
			loggingConfig.LogResponseBody = logRes
		}
	}

	// skip_paths の設定
	if skipPathsVal, ok := cfg["skip_paths"]; ok {
		if skipPaths, ok := skipPathsVal.([]any); ok {
			for _, path := range skipPaths {
				if pathStr, ok := path.(string); ok {
					loggingConfig.SkipPaths = append(loggingConfig.SkipPaths, pathStr)
				}
			}
		}
	}

	return NewLoggingMiddleware(f.logger, loggingConfig), nil
}

// createRecoveryMiddleware はリカバリーミドルウェアを生成する
func (f *Factory) createRecoveryMiddleware(cfg map[string]any) (Middleware, error) {
	recoveryConfig := RecoveryConfig{
		EnableStackTrace: false,
	}

	// enable_stack_trace の設定
	if enableVal, ok := cfg["enable_stack_trace"]; ok {
		if enable, ok := enableVal.(bool); ok {
			recoveryConfig.EnableStackTrace = enable
		}
	}

	return NewRecoveryMiddleware(f.logger, recoveryConfig), nil
}
