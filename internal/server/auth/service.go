// Package auth содержит сервис аутентификации пользователей.
package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/Gustik/trantor/internal/domain"
	"github.com/Gustik/trantor/internal/storage/postgres"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// userStorage определяет методы хранилища необходимые сервису аутентификации.
type userStorage interface {
	// CreateUser сохраняет нового пользователя в хранилище.
	CreateUser(ctx context.Context, user *domain.User) error
	// FindUserByLogin возвращает пользователя по логину.
	FindUserByLogin(ctx context.Context, login string) (*domain.User, error)
	// FindUserByID возвращает пользователя по ID.
	FindUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
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
// Заменяет user.AuthKeyHash сырого authKey на bcrypt-хеш перед сохранением.
// Возвращает ErrUserAlreadyExists если логин уже занят.
func (s *Service) Register(ctx context.Context, user *domain.User) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(user.AuthKeyHash), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrInternal, err)
	}

	user.AuthKeyHash = string(hash)

	if err := s.storage.CreateUser(ctx, user); err != nil {
		if errors.Is(err, postgres.ErrDuplicate) {
			return domain.ErrUserAlreadyExists
		}
		return fmt.Errorf("%w: %w", domain.ErrInternal, err)
	}
	return nil
}

// GetSalt возвращает argon2 salt пользователя по логину.
// Возвращает ErrUserNotFound если пользователь не найден.
func (s *Service) GetSalt(ctx context.Context, login string) ([]byte, error) {
	user, err := s.storage.FindUserByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("%w: %w", domain.ErrInternal, err)
	}
	return user.Argon2Salt, nil
}

// Login аутентифицирует пользователя и возвращает данные для расшифровки мастер-ключа.
// Возвращает ErrUserNotFound или ErrInvalidCredentials при ошибке аутентификации.
func (s *Service) Login(ctx context.Context, login string, authKey []byte) (*domain.User, error) {
	user, err := s.storage.FindUserByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("%w: %w", domain.ErrInternal, err)
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.AuthKeyHash), authKey); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	return user, nil
}

// GetUserByID возвращает пользователя по идентификатору.
func (s *Service) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, err := s.storage.FindUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("%w: %w", domain.ErrInternal, err)
	}

	return user, nil
}
