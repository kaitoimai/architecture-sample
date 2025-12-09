package redis_test

import (
	"context"
	"testing"
	"time"

	redisclient "api-gateway/pkg/redis"

	"github.com/alicebob/miniredis/v2"
)

func TestNewClient_Success(t *testing.T) {
	// miniredisでRedisモックを起動
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	tests := []struct {
		name   string
		config redisclient.Config
	}{
		{
			name: "正常な接続",
			config: redisclient.Config{
				Host:         mr.Addr(),
				DB:           0,
				PoolSize:     10,
				DialTimeout:  5 * time.Second,
				ReadTimeout:  3 * time.Second,
				WriteTimeout: 3 * time.Second,
			},
		},
		{
			name: "パスワード付き接続",
			config: redisclient.Config{
				Host:         mr.Addr(),
				Password:     "",
				DB:           1,
				PoolSize:     5,
				DialTimeout:  5 * time.Second,
				ReadTimeout:  3 * time.Second,
				WriteTimeout: 3 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := redisclient.NewClient(tt.config)
			if err != nil {
				t.Errorf("NewClient() error = %v, want nil", err)
			}
			if client == nil {
				t.Error("NewClient() returned nil client")
			}
			if client != nil {
				defer client.Close()
			}
		})
	}
}

func TestNewClient_Failure(t *testing.T) {
	tests := []struct {
		name   string
		config redisclient.Config
	}{
		{
			name: "無効なホスト",
			config: redisclient.Config{
				Host:        "invalid:9999",
				DialTimeout: 1 * time.Second,
			},
		},
		{
			name: "接続できないホスト",
			config: redisclient.Config{
				Host:        "localhost:9999",
				DialTimeout: 1 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := redisclient.NewClient(tt.config)
			if err == nil {
				t.Error("NewClient() error = nil, want error")
			}
			if client != nil {
				client.Close()
				t.Error("NewClient() returned non-nil client on error")
			}
		})
	}
}

func TestClient_Get_Success(t *testing.T) {
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

	ctx := context.Background()

	// テストデータをセット
	mr.Set("test-key", "test-value")

	tests := []struct {
		name      string
		key       string
		wantValue string
		wantErr   bool
	}{
		{
			name:      "存在するキーの取得",
			key:       "test-key",
			wantValue: "test-value",
			wantErr:   false,
		},
		{
			name:      "存在しないキーの取得",
			key:       "non-existent-key",
			wantValue: "",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := client.Get(ctx, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if value != tt.wantValue {
				t.Errorf("Get() value = %v, want %v", value, tt.wantValue)
			}
		})
	}
}

func TestClient_Set_Success(t *testing.T) {
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

	ctx := context.Background()

	tests := []struct {
		name       string
		key        string
		value      string
		expiration time.Duration
	}{
		{
			name:       "有効期限なし",
			key:        "key1",
			value:      "value1",
			expiration: 0,
		},
		{
			name:       "有効期限あり",
			key:        "key2",
			value:      "value2",
			expiration: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.Set(ctx, tt.key, tt.value, tt.expiration)
			if err != nil {
				t.Errorf("Set() error = %v, want nil", err)
				return
			}

			// 値が正しく設定されているか確認
			got, err := client.Get(ctx, tt.key)
			if err != nil {
				t.Errorf("Get() error = %v", err)
				return
			}
			if got != tt.value {
				t.Errorf("Get() = %v, want %v", got, tt.value)
			}

			// TTLの確認（有効期限ありの場合）
			if tt.expiration > 0 {
				ttl := mr.TTL(tt.key)
				if ttl <= 0 {
					t.Errorf("TTL = %v, want > 0", ttl)
				}
			}
		})
	}
}

func TestClient_Delete_Success(t *testing.T) {
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

	ctx := context.Background()

	// テストデータをセット
	mr.Set("delete-key", "delete-value")

	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{
			name:    "存在するキーの削除",
			key:     "delete-key",
			wantErr: false,
		},
		{
			name:    "存在しないキーの削除",
			key:     "non-existent-key",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.Delete(ctx, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 削除後、キーが存在しないことを確認
			exists := mr.Exists(tt.key)
			if exists {
				t.Errorf("Key %s still exists after Delete()", tt.key)
			}
		})
	}
}

func TestClient_Ping_Success(t *testing.T) {
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

	ctx := context.Background()

	err = client.Ping(ctx)
	if err != nil {
		t.Errorf("Ping() error = %v, want nil", err)
	}
}

func TestClient_Ping_Failure(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}

	client, err := redisclient.NewClient(redisclient.Config{
		Host: mr.Addr(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	// Redisサーバーを停止
	mr.Close()

	ctx := context.Background()

	err = client.Ping(ctx)
	if err == nil {
		t.Error("Ping() error = nil, want error")
	}
}

func TestClient_Close(t *testing.T) {
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

	err = client.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestClient_GetClient(t *testing.T) {
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

	underlyingClient := client.GetClient()
	if underlyingClient == nil {
		t.Error("GetClient() returned nil")
	}
}
