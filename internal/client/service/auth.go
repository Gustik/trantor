package service

import (
	"context"
	"fmt"

	"github.com/Gustik/trantor/pkg/crypto"
)

// Register регистрирует нового пользователя на сервере.
// Вычисляет auth_key и encryption_key из пароля через Argon2 на клиенте.
func (s *Service) Register(ctx context.Context, login, password string) error {
	salt, err := crypto.GenerateSalt()
	if err != nil {
		return fmt.Errorf("generate salt: %w", err)
	}

	authKey, encryptionKey := crypto.DeriveKeys(password, salt)
	masterKey, err := crypto.GenerateMasterKey()
	if err != nil {
		return fmt.Errorf("generate master key: %w", err)
	}

	encryptedMasterKey, nonce, err := crypto.Encrypt(encryptionKey, masterKey)
	if err != nil {
		return fmt.Errorf("encrypt master key: %w", err)
	}

	token, err := s.client.Register(ctx, login, authKey, encryptedMasterKey, nonce, salt)
	if err != nil {
		return fmt.Errorf("register: %w", err)
	}

	if err := s.vault.SetAuthToken(ctx, token); err != nil {
		return fmt.Errorf("save token: %w", err)
	}

	s.masterKey = masterKey

	return nil
}

// Login аутентифицирует пользователя.
// Вычисляет auth_key из пароля, получает мастер-ключ и сохраняет токен локально.
// TODO: добавить локально сохранение encryptedMasterKey чтобы каждый раз на сервер не ходить.
func (s *Service) Login(ctx context.Context, login, password string) error {
	salt, err := s.client.GetSalt(ctx, login)
	if err != nil {
		return fmt.Errorf("get salt: %w", err)
	}

	authKey, encryptionKey := crypto.DeriveKeys(password, salt)
	token, encryptedMasterKey, masterKeyNonce, err := s.client.Login(ctx, login, authKey)
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}

	masterKey, err := crypto.Decrypt(encryptionKey, masterKeyNonce, encryptedMasterKey)
	if err != nil {
		return fmt.Errorf("decrypt master key: %w", err)
	}

	if err := s.vault.SetAuthToken(ctx, token); err != nil {
		return fmt.Errorf("set token: %w", err)
	}

	s.masterKey = masterKey

	return nil
}
