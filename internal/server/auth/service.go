// Package auth содержит сервис аутентификации пользователей.
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	commondomain "github.com/Gustik/trantor/internal/common/domain"
	domain "github.com/Gustik/trantor/internal/server/domain"
	"github.com/Gustik/trantor/internal/server/storage"
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
		return fmt.Errorf("%w: %w", commondomain.ErrInternal, err)
	}

	user.AuthKeyHash = string(hash)
	user.CreatedAt = time.Now().UTC()

	if err := s.storage.CreateUser(ctx, user); err != nil {
		if errors.Is(err, storage.ErrDuplicate) {
			return commondomain.ErrUserAlreadyExists
		}
		return fmt.Errorf("%w: %w", commondomain.ErrInternal, err)
	}
	return nil
}

// GetSalt возвращает argon2 salt пользователя по логину.
// Возвращает ErrUserNotFound если пользователь не найден.
func (s *Service) GetSalt(ctx context.Context, login string) ([]byte, error) {
	user, err := s.storage.FindUserByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, commondomain.ErrUserNotFound
		}
		return nil, fmt.Errorf("%w: %w", commondomain.ErrInternal, err)
	}
	return user.Argon2Salt, nil
}

// Login аутентифицирует пользователя и возвращает данные для расшифровки мастер-ключа.
// Возвращает ErrUserNotFound или ErrInvalidCredentials при ошибке аутентификации.
func (s *Service) Login(ctx context.Context, login string, authKey []byte) (*domain.User, error) {
	user, err := s.storage.FindUserByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, commondomain.ErrUserNotFound
		}
		return nil, fmt.Errorf("%w: %w", commondomain.ErrInternal, err)
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.AuthKeyHash), authKey); err != nil {
		return nil, commondomain.ErrInvalidCredentials
	}

	return user, nil
}

// GetUserByID возвращает пользователя по идентификатору.
func (s *Service) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, err := s.storage.FindUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, commondomain.ErrUserNotFound
		}
		return nil, fmt.Errorf("%w: %w", commondomain.ErrInternal, err)
	}

	return user, nil
}
