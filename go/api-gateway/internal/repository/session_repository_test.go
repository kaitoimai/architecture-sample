package repository_test

import (
	"context"
	"testing"
	"time"

	"api-gateway/internal/repository"
	redisclient "api-gateway/pkg/redis"

	"github.com/alicebob/miniredis/v2"
)

func TestNewRedisSessionRepository(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	client, err := redisclient.NewClient(redisclient.Config{
		Host: mr.Addr(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	tests := []struct {
		name           string
		keyPrefix      string
		expectedPrefix string
	}{
		{
			name:           "カスタムプレフィックス",
			keyPrefix:      "test:",
			expectedPrefix: "test:",
		},
		{
			name:           "空のプレフィックス（デフォルト）",
			keyPrefix:      "",
			expectedPrefix: "revoke:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := repository.NewRedisSessionRepository(client, tt.keyPrefix)
			if repo == nil {
				t.Fatal("NewRedisSessionRepository returned nil")
			}

			// プレフィックスの確認（内部状態は直接確認できないため、動作で確認）
			ctx := context.Background()
			testTime := time.Now()
			userID := "test-user"

			err := repo.SetRevokedTime(ctx, userID, testTime, 10*time.Second)
			if err != nil {
				t.Errorf("SetRevokedTime() error = %v", err)
			}

			// Redisに正しいキーで保存されているか確認
			expectedKey := tt.expectedPrefix + userID
			exists := mr.Exists(expectedKey)
			if !exists {
				t.Errorf("Expected key %s to exist in Redis", expectedKey)
			}
		})
	}
}

func TestRedisSessionRepository_SetRevokedTime_Success(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	client, err := redisclient.NewClient(redisclient.Config{
		Host: mr.Addr(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	repo := repository.NewRedisSessionRepository(client, "test:")
	ctx := context.Background()

	tests := []struct {
		name        string
		userID      string
		revokedTime time.Time
		expiration  time.Duration
		wantErr     bool
		shouldSave  bool
	}{
		{
			name:        "成功: 正常な保存",
			userID:      "user123",
			revokedTime: time.Now(),
			expiration:  10 * time.Minute,
			wantErr:     false,
			shouldSave:  true,
		},
		{
			name:        "スキップ: 有効期限0",
			userID:      "user456",
			revokedTime: time.Now(),
			expiration:  0,
			wantErr:     false,
			shouldSave:  false,
		},
		{
			name:        "スキップ: 負の有効期限",
			userID:      "user789",
			revokedTime: time.Now(),
			expiration:  -1 * time.Second,
			wantErr:     false,
			shouldSave:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.SetRevokedTime(ctx, tt.userID, tt.revokedTime, tt.expiration)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetRevokedTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// キーが保存されているか確認
			key := "test:" + tt.userID
			exists := mr.Exists(key)
			if exists != tt.shouldSave {
				t.Errorf("Key existence = %v, want %v", exists, tt.shouldSave)
			}

			// 保存されている場合、TTLと値を確認
			if tt.shouldSave {
				// TTLの確認
				ttl := mr.TTL(key)
				if ttl <= 0 {
					t.Errorf("TTL = %v, want > 0", ttl)
				}

				// 値の確認
				value, _ := mr.Get(key)
				expectedValue := tt.revokedTime.Format(time.RFC3339)
				if value != expectedValue {
					t.Errorf("Value = %v, want %v", value, expectedValue)
				}
			}
		})
	}
}

func TestRedisSessionRepository_GetRevokedTime_Success(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	client, err := redisclient.NewClient(redisclient.Config{
		Host: mr.Addr(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	repo := repository.NewRedisSessionRepository(client, "test:")
	ctx := context.Background()

	// テストデータの準備
	now := time.Now().Truncate(time.Second) // RFC3339の精度に合わせる
	mr.Set("test:user1", now.Format(time.RFC3339))

	tests := []struct {
		name             string
		userID           string
		wantTime         time.Time
		wantErr          bool
		wantZero         bool
		setupInvalidData bool
	}{
		{
			name:     "成功: 失効時刻が存在",
			userID:   "user1",
			wantTime: now,
			wantErr:  false,
			wantZero: false,
		},
		{
			name:     "成功: キーが存在しない（ゼロ値）",
			userID:   "non-existent-user",
			wantTime: time.Time{},
			wantErr:  false,
			wantZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupInvalidData {
				mr.Set("test:"+tt.userID, "invalid-time-format")
			}

			gotTime, err := repo.GetRevokedTime(ctx, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRevokedTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantZero {
				if !gotTime.IsZero() {
					t.Errorf("GetRevokedTime() = %v, want zero time", gotTime)
				}
			} else if !tt.wantErr {
				if !gotTime.Equal(tt.wantTime) {
					t.Errorf("GetRevokedTime() = %v, want %v", gotTime, tt.wantTime)
				}
			}
		})
	}
}

func TestRedisSessionRepository_GetRevokedTime_ParseError(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	client, err := redisclient.NewClient(redisclient.Config{
		Host: mr.Addr(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	repo := repository.NewRedisSessionRepository(client, "test:")
	ctx := context.Background()

	// 無効な時刻フォーマットをセット
	mr.Set("test:invalid-user", "invalid-time-format")

	_, err = repo.GetRevokedTime(ctx, "invalid-user")
	if err == nil {
		t.Error("GetRevokedTime() error = nil, want parse error")
	}
}

func TestRedisSessionRepository_DeleteRevokedTime_Success(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	client, err := redisclient.NewClient(redisclient.Config{
		Host: mr.Addr(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	repo := repository.NewRedisSessionRepository(client, "test:")
	ctx := context.Background()

	// テストデータの準備
	now := time.Now()
	mr.Set("test:delete-user", now.Format(time.RFC3339))

	tests := []struct {
		name    string
		userID  string
		wantErr bool
	}{
		{
			name:    "成功: 存在するキーの削除",
			userID:  "delete-user",
			wantErr: false,
		},
		{
			name:    "成功: 存在しないキーの削除",
			userID:  "non-existent-user",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.DeleteRevokedTime(ctx, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteRevokedTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 削除後、キーが存在しないことを確認
			key := "test:" + tt.userID
			exists := mr.Exists(key)
			if exists {
				t.Errorf("Key %s still exists after DeleteRevokedTime()", key)
			}
		})
	}
}

func TestRedisSessionRepository_Integration(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	client, err := redisclient.NewClient(redisclient.Config{
		Host: mr.Addr(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	repo := repository.NewRedisSessionRepository(client, "integration:")
	ctx := context.Background()

	// シナリオ: Set → Get → Delete → Get
	userID := "integration-user"
	revokedTime := time.Now().Truncate(time.Second)
	expiration := 10 * time.Minute

	// 1. Set
	err = repo.SetRevokedTime(ctx, userID, revokedTime, expiration)
	if err != nil {
		t.Fatalf("SetRevokedTime() error = %v", err)
	}

	// 2. Get（存在確認）
	gotTime, err := repo.GetRevokedTime(ctx, userID)
	if err != nil {
		t.Fatalf("GetRevokedTime() error = %v", err)
	}
	if !gotTime.Equal(revokedTime) {
		t.Errorf("GetRevokedTime() = %v, want %v", gotTime, revokedTime)
	}

	// 3. Delete
	err = repo.DeleteRevokedTime(ctx, userID)
	if err != nil {
		t.Fatalf("DeleteRevokedTime() error = %v", err)
	}

	// 4. Get（削除確認）
	gotTime, err = repo.GetRevokedTime(ctx, userID)
	if err != nil {
		t.Fatalf("GetRevokedTime() error = %v", err)
	}
	if !gotTime.IsZero() {
		t.Errorf("GetRevokedTime() after delete = %v, want zero time", gotTime)
	}
}
