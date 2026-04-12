// Package postgres содержит реализацию хранилища на основе PostgreSQL.
package postgres

import (
	"context"
	"fmt"

	"github.com/Gustik/trantor/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Storage реализует методы работы с PostgreSQL для всех сущностей.
type Storage struct {
	db *pgxpool.Pool
}

// New создаёт новый экземпляр Storage и проверяет соединение с БД.
func New(ctx context.Context, cfg *config.DBConfig) (*Storage, error) {
	poolCfg, err := buildPoolConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("ошибка подготовки конфига пула: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания пула: %w", err)
	}

	if err = pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Storage{pool}, nil
}

// Close закрывает пул соединений с БД.
func (s *Storage) Close() {
	s.db.Close()
}

func (s *Storage) Ping(ctx context.Context) error {
	return s.db.Ping(ctx)
}

// buildPoolConfig — подготавливает конфиг для postgres
func buildPoolConfig(cfg *config.DBConfig) (*pgxpool.Config, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("неверный DSN: %w", err)
	}
	poolCfg.MaxConns = int32(cfg.MaxConns)
	poolCfg.MinConns = int32(cfg.MinConns)
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	return poolCfg, nil
}
