package postgres

import (
	"testing"
	"time"

	"github.com/Gustik/trantor/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_buildPoolConfig(t *testing.T) {
	cfg := &config.DBConfig{
		DSN:             "postgres://user:pass@localhost:5432/trantor",
		MaxConns:        20,
		MinConns:        5,
		MaxConnLifetime: 2 * time.Hour,
		MaxConnIdleTime: 15 * time.Minute,
	}

	poolCfg, err := buildPoolConfig(cfg)
	require.NoError(t, err)
	assert.Equal(t, int32(20), poolCfg.MaxConns)
	assert.Equal(t, int32(5), poolCfg.MinConns)
	assert.Equal(t, 2*time.Hour, poolCfg.MaxConnLifetime)
	assert.Equal(t, 15*time.Minute, poolCfg.MaxConnIdleTime)
}
