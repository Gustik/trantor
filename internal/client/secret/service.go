// Package secret содержит клиентский сервис управления секретами.
package secret

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Gustik/trantor/internal/client/domain"
	"github.com/Gustik/trantor/internal/client/storage"
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
	SaveSecret(ctx context.Context, r *domain.Secret) error
	// MarkSynced помечает секрет как синхронизированный с сервером.
	MarkSynced(ctx context.Context, id uuid.UUID) error
	// ListUnsynced возвращает ID секретов, ещё не отправленных на сервер.
	ListUnsynced(ctx context.Context) ([]uuid.UUID, error)
	// GetSecret возвращает локально сохранённый секрет по ID.
	GetSecret(ctx context.Context, id uuid.UUID) (*domain.Secret, error)
	// ListSecrets возвращает все локально сохранённые секреты.
	ListSecrets(ctx context.Context) ([]*domain.Secret, error)
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
		return errors.Join(domain.ErrInternal, err)
	}

	// Шифруем весь payload для отправки на сервер.
	marshaledPayload, err := json.Marshal(payload)
	if err != nil {
		return errors.Join(domain.ErrInternal, err)
	}
	serverData, serverNonce, err := crypto.Encrypt(s.masterKey, marshaledPayload)
	if err != nil {
		return errors.Join(domain.ErrInternal, err)
	}

	id := uuid.New()
	now := time.Now().UTC()

	record := &domain.Secret{
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
		return errors.Join(domain.ErrInternal, err)
	}

	token, err := s.vault.GetAuthToken(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return domain.ErrNotAuthenticated
		}
		return errors.Join(domain.ErrInternal, err)
	}

	if err := s.client.CreateSecret(ctx, token, id, serverData, serverNonce); err != nil {
		// Сервер недоступен — вернём nil, Sync() подхватит позже.
		return nil
	}

	return s.vault.MarkSynced(ctx, id)
}

// Get возвращает секрет из локального vault по ID.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*commondomain.SecretPayload, error) {
	secret, err := s.vault.GetSecret(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, domain.ErrSecretNotFound
		}
		return nil, errors.Join(domain.ErrInternal, err)
	}

	data, err := crypto.Decrypt(s.masterKey, secret.DataNonce, secret.Data)
	if err != nil {
		return nil, errors.Join(domain.ErrInternal, err)
	}

	return &commondomain.SecretPayload{
		Type:     secret.Type,
		Name:     secret.Name,
		Metadata: secret.Metadata,
		Data:     data,
	}, nil
}

// List возвращает все секреты из локального vault.
func (s *Service) List(ctx context.Context) ([]*commondomain.SecretPayload, error) {
	secrets, err := s.vault.ListSecrets(ctx)
	if err != nil {
		return nil, fmt.Errorf("list secret: %w", errors.Join(domain.ErrInternal, err))
	}

	payloads := make([]*commondomain.SecretPayload, 0, len(secrets))

	for _, secret := range secrets {
		data, err := crypto.Decrypt(s.masterKey, secret.DataNonce, secret.Data)
		if err != nil {
			return nil, errors.Join(domain.ErrInternal, err)
		}

		payloads = append(payloads, &commondomain.SecretPayload{
			Type:     secret.Type,
			Name:     secret.Name,
			Metadata: secret.Metadata,
			Data:     data,
		})
	}

	return payloads, nil
}

// Delete удаляет секрет на сервере и из локального vault.
// Сначала удаляет на сервере — если сервер недоступен, возвращает ошибку и локально не трогает.
// Это избегает ситуации когда Sync вернёт удалённый секрет обратно.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	token, err := s.vault.GetAuthToken(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return domain.ErrNotAuthenticated
		}
		return errors.Join(domain.ErrInternal, err)
	}

	if err := s.client.DeleteSecret(ctx, token, id.String()); err != nil {
		return err
	}

	if err := s.vault.DeleteSecret(ctx, id); err != nil {
		return errors.Join(domain.ErrInternal, err)
	}
	return nil
}

// Sync синхронизирует локальный vault с сервером в два шага:
// 1. Отправляет на сервер секреты с synced=false.
// 2. Забирает с сервера секреты изменённые после последней синхронизации.
func (s *Service) Sync(ctx context.Context) error {
	token, err := s.vault.GetAuthToken(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return domain.ErrNotAuthenticated
		}
		return errors.Join(domain.ErrInternal, err)
	}

	// push несинхронизированных секретов на сервер.
	unsyncedIDs, err := s.vault.ListUnsynced(ctx)
	if err != nil {
		return errors.Join(domain.ErrInternal, err)
	}
	for _, id := range unsyncedIDs {
		secret, err := s.vault.GetSecret(ctx, id)
		if err != nil {
			return errors.Join(domain.ErrInternal, err)
		}

		data, err := crypto.Decrypt(s.masterKey, secret.DataNonce, secret.Data)
		if err != nil {
			return errors.Join(domain.ErrInternal, err)
		}

		marshaled, err := json.Marshal(&commondomain.SecretPayload{
			Type:     secret.Type,
			Name:     secret.Name,
			Data:     data,
			Metadata: secret.Metadata,
		})
		if err != nil {
			return errors.Join(domain.ErrInternal, err)
		}

		serverData, serverNonce, err := crypto.Encrypt(s.masterKey, marshaled)
		if err != nil {
			return errors.Join(domain.ErrInternal, err)
		}

		if err := s.client.CreateSecret(ctx, token, id, serverData, serverNonce); err != nil {
			return err
		}
		if err := s.vault.MarkSynced(ctx, id); err != nil {
			return errors.Join(domain.ErrInternal, err)
		}
	}

	// pull изменений с сервера.
	lastSync, err := s.vault.LastSyncedAt(ctx)
	if err != nil {
		return errors.Join(domain.ErrInternal, err)
	}

	serverSecrets, err := s.client.ListSecrets(ctx, token, lastSync)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	for _, ss := range serverSecrets {
		if ss.DeletedAt != nil {
			_ = s.vault.DeleteSecret(ctx, ss.ID)
			continue
		}

		raw, err := crypto.Decrypt(s.masterKey, ss.Nonce, ss.Data)
		if err != nil {
			return errors.Join(domain.ErrInternal, err)
		}

		var payload commondomain.SecretPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return errors.Join(domain.ErrInternal, err)
		}

		encData, dataNonce, err := crypto.Encrypt(s.masterKey, payload.Data)
		if err != nil {
			return errors.Join(domain.ErrInternal, err)
		}

		record := &domain.Secret{
			ID:        ss.ID,
			Type:      payload.Type,
			Name:      payload.Name,
			Data:      encData,
			DataNonce: dataNonce,
			Metadata:  payload.Metadata,
			UpdatedAt: ss.UpdatedAt,
			Synced:    true,
		}
		if err := s.vault.SaveSecret(ctx, record); err != nil {
			return errors.Join(domain.ErrInternal, err)
		}
	}

	return s.vault.SetLastSyncedAt(ctx, now)
}
