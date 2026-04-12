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

func newSecret(userID uuid.UUID) *domain.Secret {
	now := time.Now().UTC().Truncate(time.Millisecond)
	return &domain.Secret{
		ID:        uuid.New(),
		UserID:    userID,
		Data:      []byte("encrypted-payload"),
		Nonce:     []byte("nonce-12-bytes!!"),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// createTestUser создаёт пользователя с уникальным логином и возвращает его ID.
func createTestUser(t *testing.T, ctx context.Context) uuid.UUID {
	t.Helper()
	user := newUser(uuid.NewString())
	require.NoError(t, testStore.CreateUser(ctx, user))
	return user.ID
}

func TestCreateSecret(t *testing.T) {
	ctx := context.Background()
	userID := createTestUser(t, ctx)

	t.Run("успешное создание", func(t *testing.T) {
		err := testStore.CreateSecret(ctx, newSecret(userID))
		require.NoError(t, err)
	})
}

func TestGetSecretByID(t *testing.T) {
	ctx := context.Background()
	userID := createTestUser(t, ctx)

	t.Run("найден", func(t *testing.T) {
		original := newSecret(userID)
		require.NoError(t, testStore.CreateSecret(ctx, original))

		found, err := testStore.GetSecretByID(ctx, original.ID, userID)
		require.NoError(t, err)
		assert.Equal(t, original.ID, found.ID)
		assert.Equal(t, original.UserID, found.UserID)
		assert.Equal(t, original.Data, found.Data)
		assert.Equal(t, original.Nonce, found.Nonce)
		assert.WithinDuration(t, original.CreatedAt, found.CreatedAt, time.Second)
	})

	t.Run("не найден", func(t *testing.T) {
		_, err := testStore.GetSecretByID(ctx, uuid.New(), userID)
		assert.ErrorIs(t, err, domain.ErrSecretNotFound)
	})

	t.Run("чужой секрет", func(t *testing.T) {
		otherUserID := createTestUser(t, ctx)
		secret := newSecret(otherUserID)
		require.NoError(t, testStore.CreateSecret(ctx, secret))

		_, err := testStore.GetSecretByID(ctx, secret.ID, userID)
		assert.ErrorIs(t, err, domain.ErrSecretNotFound)
	})
}

func TestListSecrets(t *testing.T) {
	ctx := context.Background()
	userID := createTestUser(t, ctx)

	secrets := make([]*domain.Secret, 3)
	for i := range secrets {
		secrets[i] = newSecret(userID)
		require.NoError(t, testStore.CreateSecret(ctx, secrets[i]))
	}

	t.Run("все секреты пользователя", func(t *testing.T) {
		list, err := testStore.ListSecrets(ctx, userID, time.Time{})
		require.NoError(t, err)
		assert.Len(t, list, 3)
	})

	t.Run("фильтр по updatedAfter", func(t *testing.T) {
		future := time.Now().UTC().Add(time.Hour)
		list, err := testStore.ListSecrets(ctx, userID, future)
		require.NoError(t, err)
		assert.Empty(t, list)
	})

	t.Run("не возвращает чужие секреты", func(t *testing.T) {
		otherUserID := createTestUser(t, ctx)
		require.NoError(t, testStore.CreateSecret(ctx, newSecret(otherUserID)))

		list, err := testStore.ListSecrets(ctx, userID, time.Time{})
		require.NoError(t, err)
		assert.Len(t, list, 3)
	})
}

func TestUpdateSecret(t *testing.T) {
	ctx := context.Background()
	userID := createTestUser(t, ctx)

	t.Run("успешное обновление", func(t *testing.T) {
		secret := newSecret(userID)
		require.NoError(t, testStore.CreateSecret(ctx, secret))

		secret.Data = []byte("updated-payload")
		secret.Nonce = []byte("new-nonce-12b!!!")
		secret.UpdatedAt = time.Now().UTC().Truncate(time.Millisecond)
		require.NoError(t, testStore.UpdateSecret(ctx, secret))

		found, err := testStore.GetSecretByID(ctx, secret.ID, userID)
		require.NoError(t, err)
		assert.Equal(t, secret.Data, found.Data)
		assert.Equal(t, secret.Nonce, found.Nonce)
	})

	t.Run("секрет не найден", func(t *testing.T) {
		ghost := newSecret(userID)
		err := testStore.UpdateSecret(ctx, ghost)
		assert.ErrorIs(t, err, domain.ErrSecretNotFound)
	})
}

func TestDeleteSecret(t *testing.T) {
	ctx := context.Background()
	userID := createTestUser(t, ctx)

	t.Run("успешное удаление", func(t *testing.T) {
		secret := newSecret(userID)
		require.NoError(t, testStore.CreateSecret(ctx, secret))

		require.NoError(t, testStore.DeleteSecret(ctx, secret.ID, userID))

		_, err := testStore.GetSecretByID(ctx, secret.ID, userID)
		assert.ErrorIs(t, err, domain.ErrSecretNotFound)
	})

	t.Run("секрет не найден", func(t *testing.T) {
		err := testStore.DeleteSecret(ctx, uuid.New(), userID)
		assert.ErrorIs(t, err, domain.ErrSecretNotFound)
	})

	t.Run("чужой секрет", func(t *testing.T) {
		otherUserID := createTestUser(t, ctx)
		secret := newSecret(otherUserID)
		require.NoError(t, testStore.CreateSecret(ctx, secret))

		err := testStore.DeleteSecret(ctx, secret.ID, userID)
		assert.ErrorIs(t, err, domain.ErrSecretNotFound)
	})
}
