// Package service содержит клиентский сервисный слой Trantor.
package service

import (
	"context"
	"time"

	"github.com/Gustik/trantor/internal/domain"
	"github.com/google/uuid"
)

// grpcClient определяет методы gRPC-клиента необходимые сервису.
type grpcClient interface {
	// Register регистрирует нового пользователя на сервере.
	Register(ctx context.Context, login string, authKey, encryptedMasterKey, masterKeyNonce, argon2Salt []byte) (token string, err error)
	// GetSalt возвращает argon2 salt пользователя с сервера.
	GetSalt(ctx context.Context, login string) ([]byte, error)
	// Login аутентифицирует пользователя и возвращает токен и зашифрованный мастер-ключ.
	Login(ctx context.Context, login string, authKey []byte) (token string, encryptedMasterKey, masterKeyNonce []byte, err error)
	// CreateSecret отправляет зашифрованный секрет на сервер.
	CreateSecret(ctx context.Context, token string, data, nonce []byte) (id string, err error)
	// GetSecret запрашивает секрет с сервера по ID.
	GetSecret(ctx context.Context, token, id string) (*domain.Secret, error)
	// ListSecrets запрашивает список секретов с сервера.
	ListSecrets(ctx context.Context, token string, updatedAfter time.Time) ([]*domain.Secret, error)
	// UpdateSecret обновляет секрет на сервере.
	UpdateSecret(ctx context.Context, token, id string, data, nonce []byte) error
	// DeleteSecret удаляет секрет на сервере.
	DeleteSecret(ctx context.Context, token, id string) error
}

// vault определяет методы локального хранилища секретов.
type vault interface {
	// SaveSecret сохраняет расшифрованный секрет локально.
	SaveSecret(ctx context.Context, payload *domain.SecretPayload, serverID uuid.UUID, updatedAt time.Time) error
	// GetSecret возвращает локально сохранённый секрет по serverID.
	GetSecret(ctx context.Context, serverID uuid.UUID) (*domain.SecretPayload, error)
	// ListSecrets возвращает все локально сохранённые секреты.
	ListSecrets(ctx context.Context) ([]*domain.SecretPayload, error)
	// DeleteSecret удаляет локально сохранённый секрет по serverID.
	DeleteSecret(ctx context.Context, serverID uuid.UUID) error
	// LastSyncedAt возвращает время последней успешной синхронизации.
	LastSyncedAt(ctx context.Context) (time.Time, error)
	// SetLastSyncedAt сохраняет время последней успешной синхронизации.
	SetLastSyncedAt(ctx context.Context, t time.Time) error
}

// Service реализует клиентскую бизнес-логику: взаимодействие с сервером и локальным хранилищем.
type Service struct {
	client grpcClient
	vault  vault
}

// New создаёт новый экземпляр Service.
func New(client grpcClient, vault vault) *Service {
	return &Service{client: client, vault: vault}
}
