// Package secret содержит сервис управления секретами пользователей.
package secret

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	commondomain "github.com/Gustik/trantor/internal/common/domain"
	"github.com/Gustik/trantor/internal/server/domain"
	"github.com/Gustik/trantor/internal/server/storage"
)

// secretStorage определяет методы хранилища необходимые сервису секретов.
type secretStorage interface {
	// CreateSecret сохраняет новый секрет в хранилище.
	CreateSecret(ctx context.Context, secret *domain.Secret) error
	// GetSecretByID возвращает секрет по ID и ID владельца.
	GetSecretByID(ctx context.Context, id, userID uuid.UUID) (*domain.Secret, error)
	// ListSecrets возвращает все секреты пользователя, изменённые после updatedAfter.
	// Если updatedAfter равен нулю — возвращаются все секреты.
	ListSecrets(ctx context.Context, userID uuid.UUID, updatedAfter time.Time) ([]*domain.Secret, error)
	// UpdateSecret обновляет существующий секрет.
	UpdateSecret(ctx context.Context, secret *domain.Secret) error
	// DeleteSecret удаляет секрет по ID и ID владельца.
	DeleteSecret(ctx context.Context, id, userID uuid.UUID) error
}

// Service реализует бизнес-логику управления секретами.
type Service struct {
	storage secretStorage
}

// New создаёт новый экземпляр Service.
func New(storage secretStorage) *Service {
	return &Service{storage: storage}
}

// Create создаёт новый секрет для пользователя.
func (s *Service) Create(ctx context.Context, secret *domain.Secret) error {
	now := time.Now().UTC()
	secret.CreatedAt = now
	secret.UpdatedAt = now

	if err := s.storage.CreateSecret(ctx, secret); err != nil {
		return fmt.Errorf("%w: %w", commondomain.ErrInternal, err)
	}
	return nil
}

// GetByID возвращает секрет по ID.
// Возвращает ErrSecretNotFound.
func (s *Service) GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Secret, error) {
	secret, err := s.storage.GetSecretByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, domain.ErrSecretNotFound
		}
		return nil, fmt.Errorf("%w: %w", commondomain.ErrInternal, err)
	}

	return secret, nil
}

// List возвращает список секретов пользователя.
// Если updatedAfter не равен нулю — возвращает только изменённые после этого времени.
func (s *Service) List(ctx context.Context, userID uuid.UUID, updatedAfter time.Time) ([]*domain.Secret, error) {
	secrets, err := s.storage.ListSecrets(ctx, userID, updatedAfter)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", commondomain.ErrInternal, err)
	}
	return secrets, nil
}

// Update обновляет существующий секрет.
// Возвращает ErrSecretNotFound.
func (s *Service) Update(ctx context.Context, secret *domain.Secret) error {
	secret.UpdatedAt = time.Now().UTC()
	if err := s.storage.UpdateSecret(ctx, secret); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return domain.ErrSecretNotFound
		}
		return fmt.Errorf("%w: %w", commondomain.ErrInternal, err)
	}
	return nil
}

// Delete удаляет секрет по ID.
// Возвращает ErrSecretNotFound.
func (s *Service) Delete(ctx context.Context, id, userID uuid.UUID) error {
	if err := s.storage.DeleteSecret(ctx, id, userID); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return domain.ErrSecretNotFound
		}
		return fmt.Errorf("%w: %w", commondomain.ErrInternal, err)
	}
	return nil
}
