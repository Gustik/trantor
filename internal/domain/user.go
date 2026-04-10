// Package domain содержит бизнес-сущности и ошибки доменного слоя.
package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// User представляет пользователя системы.
type User struct {
	// ID — уникальный идентификатор пользователя.
	ID uuid.UUID
	// Login — уникальный логин пользователя.
	Login string
	// AuthKeyHash — bcrypt-хэш auth_key, вычисленного на клиенте из пароля через Argon2.
	// Пароль пользователя на сервер не передаётся.
	AuthKeyHash string
	// EncryptedMasterKey — мастер-ключ, зашифрованный encryption_key на клиенте.
	// Сервер не может его расшифровать.
	EncryptedMasterKey []byte
	// MasterKeyNonce — nonce для расшифровки EncryptedMasterKey.
	MasterKeyNonce []byte
	// Argon2Salt — salt для деривации ключей через Argon2 на клиенте.
	Argon2Salt []byte
	// CreatedAt — время создания пользователя.
	CreatedAt time.Time
}

// Ошибки доменного слоя для пользователей.
var (
	// ErrUserNotFound возвращается когда пользователь не найден.
	ErrUserNotFound = errors.New("пользователь не найден")
	// ErrUserAlreadyExists возвращается при попытке зарегистрировать уже существующий логин.
	ErrUserAlreadyExists = errors.New("пользователь уже существует")
	// ErrInvalidCredentials возвращается при неверном auth_key во время входа.
	ErrInvalidCredentials = errors.New("неверные учётные данные")
)
