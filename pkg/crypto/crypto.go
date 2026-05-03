// Package crypto предоставляет функции шифрования AES-256-GCM и деривации ключей через Argon2.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
)

// KeySize — размер ключа AES-256 в байтах.
const KeySize = 32

// NonceSize — размер nonce для AES-256-GCM в байтах.
const NonceSize = 12

// SaltSize — размер salt для Argon2 в байтах.
const SaltSize = 32

var (
	// ErrInvalidKeySize возвращается когда размер ключа не равен KeySize.
	ErrInvalidKeySize = errors.New("invalid key size")
	// ErrDecryptionFailed возвращается когда расшифровка не удалась — неверный ключ или повреждённые данные.
	ErrDecryptionFailed = errors.New("decryption failed")
)

// DeriveKeys вычисляет auth_key и encryption_key из пароля и salt через Argon2.
// Возвращает два ключа по 32 байта:
// - authKey — первые 32 байта, используется для аутентификации на сервере
// - encryptionKey — вторые 32 байта, используется для шифрования мастер-ключа, не покидает клиент.
func DeriveKeys(password string, salt []byte) (authKey, encryptionKey []byte) {
	key := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, KeySize*2)
	authKey = make([]byte, KeySize)
	encryptionKey = make([]byte, KeySize)
	copy(authKey, key[:KeySize])
	copy(encryptionKey, key[KeySize:])
	return
}

// GenerateSalt генерирует случайный salt для Argon2.
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}
	return salt, nil
}

// GenerateMasterKey генерирует случайный мастер-ключ.
func GenerateMasterKey() ([]byte, error) {
	key := make([]byte, KeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate master key: %w", err)
	}
	return key, nil
}

// Encrypt шифрует данные ключом через AES-256-GCM.
// Возвращает зашифрованный blob и nonce.
func Encrypt(key, plaintext []byte) (ciphertext, nonce []byte, err error) {
	if len(key) != KeySize {
		return nil, nil, ErrInvalidKeySize
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("create gcm: %w", err)
	}

	nonce = make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

// Decrypt расшифровывает данные ключом и nonce через AES-256-GCM.
func Decrypt(key, nonce, ciphertext []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKeySize
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", errors.Join(ErrDecryptionFailed, err))
	}

	return plaintext, nil
}
