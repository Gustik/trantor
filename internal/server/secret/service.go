// Package secret содержит сервис управления секретами пользователей.
package secret

import (
	"context"
	"time"

	"github.com/Gustik/trantor/internal/domain"
	"github.com/google/uuid"
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
	return nil
}

// GetByID возвращает секрет по ID.
// Возвращает ErrSecretNotFound или ErrAccessDenied при ошибке доступа.
func (s *Service) GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Secret, error) {
	return nil, nil
}

// List возвращает список секретов пользователя.
// Если updatedAfter не равен нулю — возвращает только изменённые после этого времени.
func (s *Service) List(ctx context.Context, userID uuid.UUID, updatedAfter time.Time) ([]*domain.Secret, error) {
	return nil, nil
}

// Update обновляет существующий секрет.
// Возвращает ErrSecretNotFound или ErrAccessDenied при ошибке доступа.
func (s *Service) Update(ctx context.Context, secret *domain.Secret) error {
	return nil
}

// Delete удаляет секрет по ID.
// Возвращает ErrSecretNotFound или ErrAccessDenied при ошибке доступа.
func (s *Service) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return nil
}
