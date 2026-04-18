// Package secret содержит клиентский сервис управления секретами.
package secret

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	cdomain "github.com/Gustik/trantor/internal/client/domain"
	commondomain "github.com/Gustik/trantor/internal/common/domain"
	sdomain "github.com/Gustik/trantor/internal/server/domain"
	"github.com/Gustik/trantor/pkg/crypto"
)

// grpcClient определяет методы gRPC-клиента необходимые сервису секретов.
type grpcClient interface {
	// CreateSecret отправляет зашифрованный секрет на сервер.
	// id генерируется клиентом — запрос идемпотентен при повторной отправке.
	CreateSecret(ctx context.Context, token string, id uuid.UUID, data, nonce []byte) error
	// GetSecret запрашивает секрет с сервера по ID.
	GetSecret(ctx context.Context, token, id string) (*sdomain.Secret, error)
	// ListSecrets запрашивает список секретов с сервера.
	ListSecrets(ctx context.Context, token string, updatedAfter time.Time) ([]*sdomain.Secret, error)
	// UpdateSecret обновляет секрет на сервере.
	UpdateSecret(ctx context.Context, token, id string, data, nonce []byte) error
	// DeleteSecret удаляет секрет на сервере.
	DeleteSecret(ctx context.Context, token, id string) error
}

// vaultStore определяет методы локального хранилища необходимые сервису секретов.
type vaultStore interface {
	// SaveSecret сохраняет секрет локально.
	SaveSecret(ctx context.Context, r *cdomain.Secret) error
	// MarkSynced помечает секрет как синхронизированный с сервером.
	MarkSynced(ctx context.Context, id uuid.UUID) error
	// ListUnsynced возвращает ID секретов, ещё не отправленных на сервер.
	ListUnsynced(ctx context.Context) ([]uuid.UUID, error)
	// GetSecret возвращает локально сохранённый секрет по ID.
	GetSecret(ctx context.Context, id uuid.UUID) (*cdomain.Secret, error)
	// ListSecrets возвращает все локально сохранённые секреты.
	ListSecrets(ctx context.Context) ([]*cdomain.Secret, error)
	// DeleteSecret удаляет локально сохранённый секрет по ID.
	DeleteSecret(ctx context.Context, id uuid.UUID) error
	// LastSyncedAt возвращает время последней успешной синхронизации.
	LastSyncedAt(ctx context.Context) (time.Time, error)
	// SetLastSyncedAt сохраняет время последней успешной синхронизации.
	SetLastSyncedAt(ctx context.Context, t time.Time) error
	// GetAuthToken возвращает токен авторизации.
	GetAuthToken(ctx context.Context) (string, error)
}

// Service реализует клиентскую логику управления секретами.
type Service struct {
	client    grpcClient
	vault     vaultStore
	masterKey []byte
}

// New создаёт новый экземпляр Service.
// masterKey используется для шифрования секретов перед сохранением.
func New(client grpcClient, vault vaultStore, masterKey []byte) *Service {
	return &Service{client: client, vault: vault, masterKey: masterKey}
}

// Create шифрует и сохраняет новый секрет локально, затем пытается отправить на сервер.
// Локально: type/name/metadata plaintext для поиска, data зашифрована master_key.
// На сервер: весь payload зашифрован целиком — сервер ничего не знает о содержимом.
// Если сервер недоступен — секрет остаётся с synced=false и будет отправлен при следующем Sync.
func (s *Service) Create(ctx context.Context, payload *commondomain.SecretPayload) error {
	// Шифруем только Data для локального хранения.
	encryptedData, dataNonce, err := crypto.Encrypt(s.masterKey, payload.Data)
	if err != nil {
		return err
	}

	// Шифруем весь payload для отправки на сервер.
	marshaledPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	serverData, serverNonce, err := crypto.Encrypt(s.masterKey, marshaledPayload)
	if err != nil {
		return err
	}

	id := uuid.New()
	now := time.Now().UTC()

	record := &cdomain.Secret{
		ID:        id,
		Type:      payload.Type,
		Name:      payload.Name,
		Data:      encryptedData,
		DataNonce: dataNonce,
		Metadata:  payload.Metadata,
		UpdatedAt: now,
		Synced:    false,
	}
	if err := s.vault.SaveSecret(ctx, record); err != nil {
		return err
	}

	token, err := s.vault.GetAuthToken(ctx)
	if err != nil {
		return err
	}

	if err := s.client.CreateSecret(ctx, token, id, serverData, serverNonce); err != nil {
		// Сервер недоступен — вернём nil, Sync() подхватит позже.
		return nil
	}

	return s.vault.MarkSynced(ctx, id)
}

// Get возвращает секрет из локального vault по ID.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*commondomain.SecretPayload, error) {
	return nil, nil
}

// List возвращает все секреты из локального vault.
func (s *Service) List(ctx context.Context) ([]*commondomain.SecretPayload, error) {
	return nil, nil
}

// Delete удаляет секрет на сервере и из локального vault.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

// Sync синхронизирует локальный vault с сервером.
// Запрашивает только секреты изменённые после последней синхронизации.
func (s *Service) Sync(ctx context.Context) error {
	return nil
}
