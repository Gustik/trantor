package service

import (
	"context"
	"time"

	"github.com/Gustik/trantor/internal/domain"
	"github.com/google/uuid"
)

// Create шифрует и отправляет новый секрет на сервер, сохраняет локально в vault.
func (s *Service) Create(ctx context.Context, payload *domain.SecretPayload) error {
	return nil
}

// Get возвращает секрет из локального vault по serverID.
func (s *Service) Get(ctx context.Context, serverID uuid.UUID) (*domain.SecretPayload, error) {
	return nil, nil
}

// List возвращает все секреты из локального vault.
func (s *Service) List(ctx context.Context) ([]*domain.SecretPayload, error) {
	return nil, nil
}

// Delete удаляет секрет на сервере и из локального vault.
func (s *Service) Delete(ctx context.Context, serverID uuid.UUID) error {
	return nil
}

// Sync синхронизирует локальный vault с сервером.
// Запрашивает только секреты изменённые после последней синхронизации.
func (s *Service) Sync(ctx context.Context) error {
	return nil
}

// lastSyncedAt возвращает время последней синхронизации.
func (s *Service) lastSyncedAt(ctx context.Context) (time.Time, error) {
	return time.Time{}, nil
}
