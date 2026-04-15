package vault

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/mattn/go-sqlite3"

	"github.com/Gustik/trantor/internal/domain"
)

func newTestVault(t *testing.T) *Vault {
	t.Helper()
	v, err := New(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { v.Close() })
	return v
}

func testPayload() *domain.SecretPayload {
	return &domain.SecretPayload{
		Type:     domain.SecretTypeText,
		Name:     "test",
		Data:     []byte("encrypted"),
		Metadata: map[string]string{"key": "value"},
	}
}

func TestSaveAndGetSecret(t *testing.T) {
	v := newTestVault(t)
	ctx := context.Background()
	serverID := uuid.New()
	payload := testPayload()
	updatedAt := time.Now().UTC().Truncate(time.Second)

	err := v.SaveSecret(ctx, payload, serverID, updatedAt)
	require.NoError(t, err)

	got, err := v.GetSecret(ctx, serverID)
	require.NoError(t, err)
	assert.Equal(t, payload.Type, got.Type)
	assert.Equal(t, payload.Name, got.Name)
	assert.Equal(t, payload.Data, got.Data)
	assert.Equal(t, payload.Metadata, got.Metadata)
}

func TestSaveSecret_Upsert(t *testing.T) {
	v := newTestVault(t)
	ctx := context.Background()
	serverID := uuid.New()

	err := v.SaveSecret(ctx, testPayload(), serverID, time.Now())
	require.NoError(t, err)

	updated := &domain.SecretPayload{
		Type:     domain.SecretTypeText,
		Name:     "updated",
		Data:     []byte("new encrypted"),
		Metadata: map[string]string{},
	}
	err = v.SaveSecret(ctx, updated, serverID, time.Now())
	require.NoError(t, err)

	got, err := v.GetSecret(ctx, serverID)
	require.NoError(t, err)
	assert.Equal(t, "updated", got.Name)
	assert.Equal(t, []byte("new encrypted"), got.Data)
}

func TestGetSecret_NotFound(t *testing.T) {
	v := newTestVault(t)

	_, err := v.GetSecret(context.Background(), uuid.New())
	assert.ErrorIs(t, err, domain.ErrSecretNotFound)
}

func TestListSecrets(t *testing.T) {
	v := newTestVault(t)
	ctx := context.Background()

	err := v.SaveSecret(ctx, testPayload(), uuid.New(), time.Now())
	require.NoError(t, err)
	err = v.SaveSecret(ctx, testPayload(), uuid.New(), time.Now())
	require.NoError(t, err)

	list, err := v.ListSecrets(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestListSecrets_Empty(t *testing.T) {
	v := newTestVault(t)

	list, err := v.ListSecrets(context.Background())
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestDeleteSecret(t *testing.T) {
	v := newTestVault(t)
	ctx := context.Background()
	serverID := uuid.New()

	err := v.SaveSecret(ctx, testPayload(), serverID, time.Now())
	require.NoError(t, err)

	err = v.DeleteSecret(ctx, serverID)
	require.NoError(t, err)

	_, err = v.GetSecret(ctx, serverID)
	assert.ErrorIs(t, err, domain.ErrSecretNotFound)
}

func TestLastSyncedAt_NotSet(t *testing.T) {
	v := newTestVault(t)

	ts, err := v.LastSyncedAt(context.Background())
	require.NoError(t, err)
	assert.True(t, ts.IsZero())
}

func TestSetAndGetLastSyncedAt(t *testing.T) {
	v := newTestVault(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	err := v.SetLastSyncedAt(ctx, now)
	require.NoError(t, err)

	got, err := v.LastSyncedAt(ctx)
	require.NoError(t, err)
	assert.Equal(t, now, got)
}

func TestSetLastSyncedAt_Upsert(t *testing.T) {
	v := newTestVault(t)
	ctx := context.Background()

	first := time.Now().UTC().Truncate(time.Second)
	err := v.SetLastSyncedAt(ctx, first)
	require.NoError(t, err)

	second := first.Add(time.Hour)
	err = v.SetLastSyncedAt(ctx, second)
	require.NoError(t, err)

	got, err := v.LastSyncedAt(ctx)
	require.NoError(t, err)
	assert.Equal(t, second, got)
}
