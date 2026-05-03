package domain

import (
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
	AuthKeyHash string
	// EncryptedMasterKey — мастер-ключ, зашифрованный encryption_key на клиенте.
	EncryptedMasterKey []byte
	// MasterKeyNonce — nonce для расшифровки EncryptedMasterKey.
	MasterKeyNonce []byte
	// Argon2Salt — salt для деривации ключей через Argon2 на клиенте.
	Argon2Salt []byte
	// CreatedAt — время создания пользователя.
	CreatedAt time.Time
}
