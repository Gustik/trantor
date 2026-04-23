// Package domain содержит клиентские бизнес-сущности.
package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"

	commondomain "github.com/Gustik/trantor/internal/common/domain"
)

// Secret представляет секрет в локальном хранилище клиента.
// Type, Name, Metadata хранятся в открытом виде для поиска.
// Data зашифрована master_key — для расшифровки использовать DataNonce.
type Secret struct {
	// ID — уникальный идентификатор, назначается клиентом (UUID v4).
	ID uuid.UUID
	// Type — тип секрета.
	Type commondomain.SecretType
	// Name — человекочитаемое имя секрета.
	Name string
	// Data — AES-GCM(masterKey, SecretPayload.Data). Хранится зашифрованным.
	Data []byte
	// DataNonce — nonce для расшифровки Data.
	DataNonce []byte
	// Metadata — произвольные метаданные в формате ключ-значение (plaintext).
	Metadata map[string]string
	// UpdatedAt — время последнего обновления секрета.
	UpdatedAt time.Time
	// Synced — true если секрет отправлен на сервер.
	Synced bool
}

// Ошибки доменного слоя для секретов.
var (
	// ErrSecretNotFound возвращается когда секрет не найден.
	ErrSecretNotFound = errors.New("секрет не найден")
)
