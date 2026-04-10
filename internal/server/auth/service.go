// Package auth содержит сервис аутентификации пользователей.
package auth

import (
	"context"

	"github.com/Gustik/trantor/internal/domain"
	"github.com/google/uuid"
)

// userStorage определяет методы хранилища необходимые сервису аутентификации.
type userStorage interface {
	// CreateUser сохраняет нового пользователя в хранилище.
	CreateUser(ctx context.Context, user *domain.User) error
	// FindUserByLogin возвращает пользователя по логину.
	FindUserByLogin(ctx context.Context, login string) (*domain.User, error)
}

// Service реализует бизнес-логику аутентификации пользователей.
type Service struct {
	storage userStorage
}

// New создаёт новый экземпляр Service.
func New(storage userStorage) *Service {
	return &Service{storage: storage}
}

// Register регистрирует нового пользователя.
// Возвращает ErrUserAlreadyExists если логин уже занят.
func (s *Service) Register(ctx context.Context, user *domain.User) error {
	return nil
}

// GetSalt возвращает argon2 salt пользователя по логину.
// Возвращает ErrUserNotFound если пользователь не найден.
func (s *Service) GetSalt(ctx context.Context, login string) ([]byte, error) {
	return nil, nil
}

// Login аутентифицирует пользователя и возвращает данные для расшифровки мастер-ключа.
// Возвращает ErrUserNotFound или ErrInvalidCredentials при ошибке аутентификации.
func (s *Service) Login(ctx context.Context, login string, authKey []byte) (*domain.User, error) {
	return nil, nil
}

// GetUserByID возвращает пользователя по идентификатору.
func (s *Service) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return nil, nil
}
