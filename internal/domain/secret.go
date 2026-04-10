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

// Secret представляет зашифрованный секрет пользователя.
// Поле Data содержит зашифрованный blob — сервер не знает его содержимого.
type Secret struct {
	// ID — уникальный идентификатор секрета.
	ID uuid.UUID
	// UserID — идентификатор владельца секрета.
	UserID uuid.UUID
	// Data — зашифрованный blob: AES-GCM(master_key, SecretPayload).
	Data []byte
	// Nonce — nonce для расшифровки Data.
	Nonce []byte
	// CreatedAt — время создания секрета.
	CreatedAt time.Time
	// UpdatedAt — время последнего обновления секрета.
	UpdatedAt time.Time
}

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

// Ошибки доменного слоя для секретов.
var (
	// ErrSecretNotFound возвращается когда секрет не найден.
	ErrSecretNotFound = errors.New("секрет не найден")
	// ErrAccessDenied возвращается при попытке получить доступ к чужому секрету.
	ErrAccessDenied = errors.New("доступ запрещён")
)
