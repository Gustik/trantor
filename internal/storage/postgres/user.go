package postgres

import (
	"context"

	"github.com/Gustik/trantor/internal/domain"
)

// CreateUser сохраняет нового пользователя в БД.
// Возвращает ErrUserAlreadyExists если логин уже занят.
func (s *Storage) CreateUser(ctx context.Context, user *domain.User) error {
	return nil
}

// FindUserByLogin возвращает пользователя по логину.
// Возвращает ErrUserNotFound если пользователь не найден.
func (s *Storage) FindUserByLogin(ctx context.Context, login string) (*domain.User, error) {
	return nil, nil
}
