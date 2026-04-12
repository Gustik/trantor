//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Gustik/trantor/internal/domain"
)

func newUserForAuth(login string, authKey []byte) *domain.User {
	return &domain.User{
		ID:                 uuid.New(),
		Login:              login,
		AuthKeyHash:        string(authKey),
		EncryptedMasterKey: []byte("encrypted-master-key"),
		MasterKeyNonce:     []byte("nonce-12-bytes!!"),
		Argon2Salt:         []byte("argon2-salt-16b!"),
		CreatedAt:          time.Now().UTC().Truncate(time.Millisecond),
	}
}

func TestAuthService_Register(t *testing.T) {
	ctx := context.Background()

	t.Run("успешная регистрация", func(t *testing.T) {
		user := newUserForAuth("auth_alice", []byte("raw-auth-key-32-bytes-long!!!!!!"))
		err := testAuthService.Register(ctx, user)
		require.NoError(t, err)
		// AuthKeyHash должен стать bcrypt-хешем, не сырым ключом
		assert.NotEqual(t, "raw-auth-key-32-bytes-long!!!!!!", user.AuthKeyHash)
		assert.Contains(t, user.AuthKeyHash, "$2a$")
	})

	t.Run("дублирующий логин", func(t *testing.T) {
		user := newUserForAuth("auth_bob", []byte("raw-auth-key-32-bytes-long!!!!!!"))
		require.NoError(t, testAuthService.Register(ctx, user))

		duplicate := newUserForAuth("auth_bob", []byte("other-auth-key-32-bytes-long!!!!"))
		err := testAuthService.Register(ctx, duplicate)
		assert.ErrorIs(t, err, domain.ErrUserAlreadyExists)
	})
}

func TestAuthService_GetSalt(t *testing.T) {
	ctx := context.Background()

	t.Run("найден", func(t *testing.T) {
		user := newUserForAuth("auth_charlie", []byte("raw-auth-key-32-bytes-long!!!!!!"))
		require.NoError(t, testAuthService.Register(ctx, user))

		salt, err := testAuthService.GetSalt(ctx, "auth_charlie")
		require.NoError(t, err)
		assert.Equal(t, user.Argon2Salt, salt)
	})

	t.Run("не найден", func(t *testing.T) {
		_, err := testAuthService.GetSalt(ctx, "auth_nonexistent")
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}

func TestAuthService_Login(t *testing.T) {
	ctx := context.Background()
	authKey := []byte("raw-auth-key-32-bytes-long!!!!!!")

	user := newUserForAuth("auth_dave", authKey)
	require.NoError(t, testAuthService.Register(ctx, user))

	t.Run("успешный логин", func(t *testing.T) {
		got, err := testAuthService.Login(ctx, "auth_dave", authKey)
		require.NoError(t, err)
		assert.Equal(t, user.ID, got.ID)
		assert.Equal(t, user.EncryptedMasterKey, got.EncryptedMasterKey)
	})

	t.Run("неверный authKey", func(t *testing.T) {
		_, err := testAuthService.Login(ctx, "auth_dave", []byte("wrong-key"))
		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})

	t.Run("пользователь не найден", func(t *testing.T) {
		_, err := testAuthService.Login(ctx, "auth_nobody", authKey)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}
