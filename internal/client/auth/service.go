// Package auth содержит клиентский сервис аутентификации.
package auth

import (
	"context"
	"fmt"

	commondomain "github.com/Gustik/trantor/internal/common/domain"
	"github.com/Gustik/trantor/pkg/crypto"
)

// grpcClient определяет методы gRPC-клиента необходимые сервису аутентификации.
type grpcClient interface {
	// Register регистрирует нового пользователя на сервере.
	Register(ctx context.Context, login string, authKey, encryptedMasterKey, masterKeyNonce, argon2Salt []byte) (token string, err error)
	// GetSalt возвращает argon2 salt пользователя с сервера.
	GetSalt(ctx context.Context, login string) ([]byte, error)
	// Login аутентифицирует пользователя и возвращает токен и зашифрованный мастер-ключ.
	Login(ctx context.Context, login string, authKey []byte) (token string, encryptedMasterKey, masterKeyNonce []byte, err error)
}

// vaultStore определяет методы локального хранилища необходимые сервису аутентификации.
type vaultStore interface {
	// SetAuthToken сохраняет токен авторизации.
	SetAuthToken(ctx context.Context, token string) error
	// GetAuthToken возвращает токен авторизации.
	GetAuthToken(ctx context.Context) (string, error)
}

// Service реализует клиентскую логику аутентификации.
type Service struct {
	client grpcClient
	vault  vaultStore
}

// New создаёт новый экземпляр Service.
func New(client grpcClient, vault vaultStore) *Service {
	return &Service{client: client, vault: vault}
}

// Register регистрирует нового пользователя на сервере.
// Вычисляет auth_key и encryption_key из пароля через Argon2 на клиенте.
// Возвращает master_key для последующего шифрования секретов.
func (s *Service) Register(ctx context.Context, login, password string) (masterKey []byte, err error) {
	salt, err := crypto.GenerateSalt()
	if err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}

	authKey, encryptionKey := crypto.DeriveKeys(password, salt)
	masterKey, err = crypto.GenerateMasterKey()
	if err != nil {
		return nil, fmt.Errorf("generate master key: %w", err)
	}

	encryptedMasterKey, nonce, err := crypto.Encrypt(encryptionKey, masterKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt master key: %w", err)
	}

	token, err := s.client.Register(ctx, login, authKey, encryptedMasterKey, nonce, salt)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", commondomain.ErrInternal, err)
	}

	if err := s.vault.SetAuthToken(ctx, token); err != nil {
		return nil, fmt.Errorf("save token: %w", err)
	}

	return masterKey, nil
}

// Login аутентифицирует пользователя.
// Вычисляет auth_key из пароля, получает и расшифровывает мастер-ключ с сервера.
// Возвращает master_key для последующего шифрования секретов.
// TODO: добавить локальное сохранение encryptedMasterKey чтобы каждый раз на сервер не ходить.
func (s *Service) Login(ctx context.Context, login, password string) (masterKey []byte, err error) {
	salt, err := s.client.GetSalt(ctx, login)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", commondomain.ErrInternal, err)
	}

	authKey, encryptionKey := crypto.DeriveKeys(password, salt)
	token, encryptedMasterKey, masterKeyNonce, err := s.client.Login(ctx, login, authKey)
	if err != nil {
		return nil, err
	}

	masterKey, err = crypto.Decrypt(encryptionKey, masterKeyNonce, encryptedMasterKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt master key: %w", err)
	}

	if err := s.vault.SetAuthToken(ctx, token); err != nil {
		return nil, fmt.Errorf("set token: %w", err)
	}

	return masterKey, nil
}
