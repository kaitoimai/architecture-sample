package config

import (
	"os"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		envs        map[string]string
		expected    uint
		shouldError bool
	}{
		{
			name:     "デフォルト値が使用される",
			envs:     map[string]string{},
			expected: 8080,
		},
		{
			name: "環境変数から値を読み込む",
			envs: map[string]string{
				"PORT": "3000",
			},
			expected: 3000,
		},
		{
			name: "不正な値の場合エラーを返す",
			envs: map[string]string{
				"PORT": "invalid",
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()

			for k, v := range tt.envs {
				os.Setenv(k, v)
			}

			cfg, err := New()
			if tt.shouldError {
				if err == nil {
					t.Error("期待したエラーが発生しなかった")
				}
				return
			}

			if err != nil {
				t.Errorf("予期しないエラー: %v", err)
				return
			}

			if cfg.Port != tt.expected {
				t.Errorf("Port = %v, want %v", cfg.Port, tt.expected)
			}
		})
	}
}

func TestGetDefaultUintEnv(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		defaultVal  uint
		envValue    string
		expected    uint
		shouldError bool
	}{
		{
			name:       "環境変数が設定されていない場合",
			key:        "TEST_KEY",
			defaultVal: 100,
			envValue:   "",
			expected:   100,
		},
		{
			name:       "正常な値が設定されている場合",
			key:        "TEST_KEY",
			defaultVal: 100,
			envValue:   "200",
			expected:   200,
		},
		{
			name:        "不正な値が設定されている場合",
			key:         "TEST_KEY",
			defaultVal:  100,
			envValue:    "invalid",
			shouldError: true,
		},
		{
			name:        "負の値が設定されている場合",
			key:         "TEST_KEY",
			defaultVal:  100,
			envValue:    "-1",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()

			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
			}

			result, err := getDefaultUintEnv(tt.key, tt.defaultVal)
			if tt.shouldError {
				if err == nil {
					t.Error("期待したエラーが発生しなかった")
				}
				return
			}

			if err != nil {
				t.Errorf("予期しないエラー: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("result = %v, want %v", result, tt.expected)
			}
		})
	}
}
