// Package domain содержит клиентские бизнес-сущности.
package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// SecretType определяет тип хранимого секрета.
type SecretType string

const (
	// SecretTypeLoginPassword — пара логин/пароль.
	SecretTypeLoginPassword SecretType = "login_password"
	// SecretTypeText — произвольные текстовые данные.
	SecretTypeText SecretType = "text"
	// SecretTypeBinary — произвольные бинарные данные.
	SecretTypeBinary SecretType = "binary"
	// SecretTypeBankCard — данные банковской карты.
	SecretTypeBankCard SecretType = "bank_card"
)

// SecretPayload — структура данных секрета, которую шифрует клиент перед отправкой на сервер.
// Сервер никогда не видит это в расшифрованном виде.
type SecretPayload struct {
	// Type — тип секрета.
	Type SecretType
	// Name — человекочитаемое имя секрета, например "mysite.com".
	Name string
	// Data — сами данные (логин+пароль, текст, бинарные данные, карта).
	Data []byte
	// Metadata — произвольные метаданные в формате ключ-значение.
	Metadata map[string]string
}

// Secret представляет секрет в локальном хранилище клиента.
// Type, Name, Metadata хранятся в открытом виде для поиска.
// Data зашифрована master_key — для расшифровки использовать DataNonce.
type Secret struct {
	// ID — уникальный идентификатор, назначается клиентом (UUID v4).
	ID uuid.UUID
	// Type — тип секрета.
	Type SecretType
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
