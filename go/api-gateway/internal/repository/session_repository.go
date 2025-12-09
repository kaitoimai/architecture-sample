package repository

import (
	"context"
	"fmt"
	"time"

	redisclient "api-gateway/pkg/redis"
)

// SessionRepository はセッション管理のリポジトリインターフェース
type SessionRepository interface {
	// SetRevokedTime はユーザーのJWT失効時刻を設定する
	SetRevokedTime(ctx context.Context, userID string, revokedTime time.Time, expiration time.Duration) error

	// GetRevokedTime はユーザーのJWT失効時刻を取得する
	// 失効時刻が設定されていない場合はゼロ値を返す
	GetRevokedTime(ctx context.Context, userID string) (time.Time, error)

	// DeleteRevokedTime はユーザーのJWT失効時刻を削除する
	DeleteRevokedTime(ctx context.Context, userID string) error
}

// RedisSessionRepository はRedisを使用したセッションリポジトリの実装
type RedisSessionRepository struct {
	client    *redisclient.Client
	keyPrefix string
}

// NewRedisSessionRepository は新しいRedisSessionRepositoryを作成する
func NewRedisSessionRepository(client *redisclient.Client, keyPrefix string) *RedisSessionRepository {
	if keyPrefix == "" {
		keyPrefix = "revoke:" // デフォルトプレフィックス
	}
	return &RedisSessionRepository{
		client:    client,
		keyPrefix: keyPrefix,
	}
}

// SetRevokedTime はユーザーのJWT失効時刻を設定する
func (r *RedisSessionRepository) SetRevokedTime(ctx context.Context, userID string, revokedTime time.Time, expiration time.Duration) error {
	// 既に有効期限が切れている場合は保存しない
	if expiration <= 0 {
		return nil
	}

	key := r.makeKey(userID)
	value := revokedTime.Format(time.RFC3339)

	if err := r.client.Set(ctx, key, value, expiration); err != nil {
		return fmt.Errorf("failed to set revoked time for user %s: %w", userID, err)
	}

	return nil
}

// GetRevokedTime はユーザーのJWT失効時刻を取得する
func (r *RedisSessionRepository) GetRevokedTime(ctx context.Context, userID string) (time.Time, error) {
	key := r.makeKey(userID)

	value, err := r.client.Get(ctx, key)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get revoked time for user %s: %w", userID, err)
	}

	// キーが存在しない場合はゼロ値を返す
	if value == "" {
		return time.Time{}, nil
	}

	revokedTime, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse revoked time for user %s: %w", userID, err)
	}

	return revokedTime, nil
}

// DeleteRevokedTime はユーザーのJWT失効時刻を削除する
func (r *RedisSessionRepository) DeleteRevokedTime(ctx context.Context, userID string) error {
	key := r.makeKey(userID)

	if err := r.client.Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to delete revoked time for user %s: %w", userID, err)
	}

	return nil
}

// makeKey はユーザーIDからRedisキーを生成する
func (r *RedisSessionRepository) makeKey(userID string) string {
	return fmt.Sprintf("%s%s", r.keyPrefix, userID)
}
