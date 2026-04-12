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
	pgstore "github.com/Gustik/trantor/internal/storage/postgres"
)

func newUser(login string) *domain.User {
	return &domain.User{
		ID:                 uuid.New(),
		Login:              login,
		AuthKeyHash:        "argon2id$...",
		EncryptedMasterKey: []byte("encrypted-master-key"),
		MasterKeyNonce:     []byte("nonce-12-bytes!!"),
		Argon2Salt:         []byte("salt-16-bytes!!!"),
		CreatedAt:          time.Now().UTC().Truncate(time.Millisecond),
	}
}

func TestCreateUser(t *testing.T) {
	ctx := context.Background()

	t.Run("успешное создание", func(t *testing.T) {
		err := testStore.CreateUser(ctx, newUser("alice"))
		require.NoError(t, err)
	})

	t.Run("ошибка при создании дубликата", func(t *testing.T) {
		err := testStore.CreateUser(ctx, newUser("alice"))
		assert.ErrorIs(t, err, pgstore.ErrDuplicate)
	})
}

func TestFindUserByLogin(t *testing.T) {
	ctx := context.Background()

	t.Run("найден", func(t *testing.T) {
		original := newUser("charlie")
		require.NoError(t, testStore.CreateUser(ctx, original))

		found, err := testStore.FindUserByLogin(ctx, "charlie")
		require.NoError(t, err)
		assert.Equal(t, original.ID, found.ID)
		assert.Equal(t, original.Login, found.Login)
		assert.Equal(t, original.AuthKeyHash, found.AuthKeyHash)
		assert.Equal(t, original.EncryptedMasterKey, found.EncryptedMasterKey)
		assert.Equal(t, original.MasterKeyNonce, found.MasterKeyNonce)
		assert.Equal(t, original.Argon2Salt, found.Argon2Salt)
		assert.WithinDuration(t, original.CreatedAt, found.CreatedAt, time.Second)
	})

	t.Run("не найден", func(t *testing.T) {
		_, err := testStore.FindUserByLogin(ctx, "nonexistent")
		assert.ErrorIs(t, err, pgstore.ErrNotFound)
	})
}
