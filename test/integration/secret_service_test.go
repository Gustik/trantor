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

func newSecretForService(userID uuid.UUID) *domain.Secret {
	return &domain.Secret{
		ID:     uuid.New(),
		UserID: userID,
		Data:   []byte("encrypted-payload"),
		Nonce:  []byte("nonce-12-bytes!!"),
	}
}

func createUserForSecretTest(t *testing.T, ctx context.Context, login string) uuid.UUID {
	t.Helper()
	user := newUserForAuth(login, []byte("raw-auth-key-32-bytes-long!!!!!!"))
	require.NoError(t, testAuthService.Register(ctx, user))
	return user.ID
}

func TestSecretService_Create(t *testing.T) {
	ctx := context.Background()
	userID := createUserForSecretTest(t, ctx, "svc_secret_alice")

	t.Run("успешное создание", func(t *testing.T) {
		s := newSecretForService(userID)
		err := testSecretService.Create(ctx, s)
		require.NoError(t, err)
		assert.False(t, s.CreatedAt.IsZero())
		assert.False(t, s.UpdatedAt.IsZero())
		assert.Equal(t, s.CreatedAt, s.UpdatedAt)
	})
}

func TestSecretService_GetByID(t *testing.T) {
	ctx := context.Background()
	userID := createUserForSecretTest(t, ctx, "svc_secret_bob")

	t.Run("найден", func(t *testing.T) {
		s := newSecretForService(userID)
		require.NoError(t, testSecretService.Create(ctx, s))

		found, err := testSecretService.GetByID(ctx, s.ID, userID)
		require.NoError(t, err)
		assert.Equal(t, s.ID, found.ID)
		assert.Equal(t, s.Data, found.Data)
	})

	t.Run("не найден", func(t *testing.T) {
		_, err := testSecretService.GetByID(ctx, uuid.New(), userID)
		assert.ErrorIs(t, err, domain.ErrSecretNotFound)
	})
}

func TestSecretService_List(t *testing.T) {
	ctx := context.Background()
	userID := createUserForSecretTest(t, ctx, "svc_secret_charlie")

	for range 3 {
		require.NoError(t, testSecretService.Create(ctx, newSecretForService(userID)))
	}

	t.Run("все секреты", func(t *testing.T) {
		list, err := testSecretService.List(ctx, userID, time.Time{})
		require.NoError(t, err)
		assert.Len(t, list, 3)
	})

	t.Run("фильтр по времени", func(t *testing.T) {
		list, err := testSecretService.List(ctx, userID, time.Now().Add(time.Hour))
		require.NoError(t, err)
		assert.Empty(t, list)
	})
}

func TestSecretService_Update(t *testing.T) {
	ctx := context.Background()
	userID := createUserForSecretTest(t, ctx, "svc_secret_dave")

	t.Run("успешное обновление", func(t *testing.T) {
		s := newSecretForService(userID)
		require.NoError(t, testSecretService.Create(ctx, s))

		s.Data = []byte("updated-payload")
		require.NoError(t, testSecretService.Update(ctx, s))

		found, err := testSecretService.GetByID(ctx, s.ID, userID)
		require.NoError(t, err)
		assert.Equal(t, []byte("updated-payload"), found.Data)
		assert.True(t, found.UpdatedAt.After(found.CreatedAt))
	})

	t.Run("не найден", func(t *testing.T) {
		ghost := newSecretForService(userID)
		err := testSecretService.Update(ctx, ghost)
		assert.ErrorIs(t, err, domain.ErrSecretNotFound)
	})
}

func TestSecretService_Delete(t *testing.T) {
	ctx := context.Background()
	userID := createUserForSecretTest(t, ctx, "svc_secret_eve")

	t.Run("успешное удаление", func(t *testing.T) {
		s := newSecretForService(userID)
		require.NoError(t, testSecretService.Create(ctx, s))

		require.NoError(t, testSecretService.Delete(ctx, s.ID, userID))

		_, err := testSecretService.GetByID(ctx, s.ID, userID)
		assert.ErrorIs(t, err, domain.ErrSecretNotFound)
	})

	t.Run("не найден", func(t *testing.T) {
		err := testSecretService.Delete(ctx, uuid.New(), userID)
		assert.ErrorIs(t, err, domain.ErrSecretNotFound)
	})
}
