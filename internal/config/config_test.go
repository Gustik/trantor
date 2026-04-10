package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func unsetenv(t *testing.T, key string) {
	t.Helper()
	old, ok := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unsetenv %s: %v", key, err)
	}
	if ok {
		t.Cleanup(func() { os.Setenv(key, old) })
	}
}

func TestLoadServer(t *testing.T) {
	t.Run("все переменные заданы", func(t *testing.T) {
		t.Setenv("TRANTOR_GRPC", ":9090")
		t.Setenv("TRANTOR_DSN", "postgres://user:pass@localhost:5432/trantor?sslmode=disable")
		t.Setenv("TRANTOR_JWT_SECRET", "super-secret-key")

		cfg, err := LoadServer()
		require.NoError(t, err)
		assert.Equal(t, ":9090", cfg.GRPC)
		assert.Equal(t, "postgres://user:pass@localhost:5432/trantor?sslmode=disable", cfg.DSN)
		assert.Equal(t, "super-secret-key", cfg.JWTSecret)
	})

	t.Run("значение по умолчанию для GRPC", func(t *testing.T) {
		unsetenv(t, "TRANTOR_GRPC")
		t.Setenv("TRANTOR_DSN", "postgres://user:pass@localhost:5432/trantor?sslmode=disable")
		t.Setenv("TRANTOR_JWT_SECRET", "super-secret-key")

		cfg, err := LoadServer()
		require.NoError(t, err)
		assert.Equal(t, ":50051", cfg.GRPC)
	})

	t.Run("DSN не задан", func(t *testing.T) {
		t.Setenv("TRANTOR_GRPC", ":9090")
		unsetenv(t, "TRANTOR_DSN")
		t.Setenv("TRANTOR_JWT_SECRET", "super-secret-key")

		_, err := LoadServer()
		assert.Error(t, err)
	})

	t.Run("JWTSecret не задан", func(t *testing.T) {
		t.Setenv("TRANTOR_GRPC", ":9090")
		t.Setenv("TRANTOR_DSN", "postgres://user:pass@localhost:5432/trantor?sslmode=disable")
		unsetenv(t, "TRANTOR_JWT_SECRET")

		_, err := LoadServer()
		assert.Error(t, err)
	})
}

func TestLoadClient(t *testing.T) {
	t.Run("все переменные заданы", func(t *testing.T) {
		t.Setenv("TRANTOR_SERVER_ADDR", "myserver:50051")
		t.Setenv("TRANTOR_VAULT_PATH", "/tmp/vault.db")

		cfg, err := LoadClient()
		require.NoError(t, err)
		assert.Equal(t, "myserver:50051", cfg.ServerAddr)
		assert.Equal(t, "/tmp/vault.db", cfg.VaultPath)
	})

	t.Run("значения по умолчанию", func(t *testing.T) {
		unsetenv(t, "TRANTOR_SERVER_ADDR")
		unsetenv(t, "TRANTOR_VAULT_PATH")

		cfg, err := LoadClient()
		require.NoError(t, err)
		assert.Equal(t, "localhost:50051", cfg.ServerAddr)
		assert.Equal(t, "~/.trantor/vault.db", cfg.VaultPath)
	})
}
