// Package vault содержит локальное хранилище секретов на основе SQLite.
package vault

import (
	"context"
	"time"

	"github.com/Gustik/trantor/internal/domain"
	"github.com/google/uuid"
)

// Vault реализует локальное хранилище секретов на основе SQLite.
type Vault struct {
	path string
}

// New открывает или создаёт локальное хранилище по указанному пути.
func New(path string) (*Vault, error) {
	return nil, nil
}

// SaveSecret сохраняет расшифрованный секрет локально.
// metadata хранится в открытом виде, data — зашифрован мастер-ключом.
func (v *Vault) SaveSecret(ctx context.Context, payload *domain.SecretPayload, serverID uuid.UUID, updatedAt time.Time) error {
	return nil
}

// GetSecret возвращает локально сохранённый секрет по serverID.
func (v *Vault) GetSecret(ctx context.Context, serverID uuid.UUID) (*domain.SecretPayload, error) {
	return nil, nil
}

// ListSecrets возвращает все локально сохранённые секреты.
func (v *Vault) ListSecrets(ctx context.Context) ([]*domain.SecretPayload, error) {
	return nil, nil
}

// DeleteSecret удаляет локально сохранённый секрет по serverID.
func (v *Vault) DeleteSecret(ctx context.Context, serverID uuid.UUID) error {
	return nil
}

// LastSyncedAt возвращает время последней успешной синхронизации с сервером.
func (v *Vault) LastSyncedAt(ctx context.Context) (time.Time, error) {
	return time.Time{}, nil
}

// SetLastSyncedAt сохраняет время последней успешной синхронизации с сервером.
func (v *Vault) SetLastSyncedAt(ctx context.Context, t time.Time) error {
	return nil
}
