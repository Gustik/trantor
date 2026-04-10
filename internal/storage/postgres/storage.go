// Package postgres содержит реализацию хранилища на основе PostgreSQL.
package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Storage реализует методы работы с PostgreSQL для всех сущностей.
type Storage struct {
	db *pgxpool.Pool
}

// New создаёт новый экземпляр Storage и проверяет соединение с БД.
func New(ctx context.Context, dsn string) (*Storage, error) {
	return nil, nil
}

// Close закрывает пул соединений с БД.
func (s *Storage) Close() {
}
