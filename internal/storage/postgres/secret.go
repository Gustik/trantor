package postgres

import (
	"context"
	"time"

	"github.com/Gustik/trantor/internal/domain"
	"github.com/google/uuid"
)

// CreateSecret сохраняет новый секрет в БД.
func (s *Storage) CreateSecret(ctx context.Context, secret *domain.Secret) error {
	return nil
}

// GetSecretByID возвращает секрет по ID и ID владельца.
// Возвращает ErrSecretNotFound если секрет не найден.
// Возвращает ErrAccessDenied если секрет принадлежит другому пользователю.
func (s *Storage) GetSecretByID(ctx context.Context, id, userID uuid.UUID) (*domain.Secret, error) {
	return nil, nil
}

// ListSecrets возвращает все секреты пользователя изменённые после updatedAfter.
// Если updatedAfter равен нулю — возвращаются все секреты пользователя.
func (s *Storage) ListSecrets(ctx context.Context, userID uuid.UUID, updatedAfter time.Time) ([]*domain.Secret, error) {
	return nil, nil
}

// UpdateSecret обновляет существующий секрет в БД.
func (s *Storage) UpdateSecret(ctx context.Context, secret *domain.Secret) error {
	return nil
}

// DeleteSecret удаляет секрет по ID и ID владельца.
func (s *Storage) DeleteSecret(ctx context.Context, id, userID uuid.UUID) error {
	return nil
}
