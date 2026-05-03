// Package domain содержит серверные бизнес-сущности.
package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrSecretNotFound возвращается когда секрет не найден.
var ErrSecretNotFound = errors.New("секрет не найден")

// Secret представляет зашифрованный секрет пользователя на сервере.
// Data содержит зашифрованный blob — сервер не знает его содержимого.
// Если DeletedAt != nil — секрет мягко удалён: Data и Nonce обнулены.
type Secret struct {
	// ID — уникальный идентификатор, назначается клиентом (UUID v4).
	ID uuid.UUID
	// UserID — идентификатор владельца секрета.
	UserID uuid.UUID
	// Data — зашифрованный blob: AES-GCM(master_key, SecretPayload). Nil если удалён.
	Data []byte
	// Nonce — nonce для расшифровки Data. Nil если удалён.
	Nonce []byte
	// CreatedAt — время создания секрета.
	CreatedAt time.Time
	// UpdatedAt — время последнего обновления секрета.
	UpdatedAt time.Time
	// DeletedAt — время мягкого удаления. Nil если секрет активен.
	DeletedAt *time.Time
}
