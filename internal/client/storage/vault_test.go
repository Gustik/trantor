package storage

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Gustik/trantor/internal/client/domain"
	"github.com/Gustik/trantor/pkg/crypto"
)

func newTestVault(t *testing.T) *Vault {
	t.Helper()
	v, err := New(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Errorf("close vault: %v", err)
		}
	})
	return v
}

var testMasterKey = make([]byte, crypto.KeySize)

func testSecret(t *testing.T) *domain.Secret {
	t.Helper()
	data := []byte("secret-data")
	encryptedData, nonce, err := crypto.Encrypt(testMasterKey, data)
	require.NoError(t, err)
	return &domain.Secret{
		ID:        uuid.New(),
		Type:      domain.SecretTypeText,
		Name:      "test",
		Data:      encryptedData,
		DataNonce: nonce,
		Metadata:  map[string]string{"key": "value"},
		UpdatedAt: time.Now().UTC().Truncate(time.Second),
		Synced:    true,
	}
}

func TestSaveAndGetSecret(t *testing.T) {
	v := newTestVault(t)
	ctx := context.Background()
	r := testSecret(t)

	err := v.SaveSecret(ctx, r)
	require.NoError(t, err)

	got, err := v.GetSecret(ctx, r.ID)
	require.NoError(t, err)
	assert.Equal(t, r.ID, got.ID)
	assert.Equal(t, r.Type, got.Type)
	assert.Equal(t, r.Name, got.Name)
	assert.Equal(t, r.Data, got.Data)
	assert.Equal(t, r.DataNonce, got.DataNonce)
	assert.Equal(t, r.Metadata, got.Metadata)
	assert.Equal(t, r.UpdatedAt, got.UpdatedAt)
	assert.Equal(t, r.Synced, got.Synced)
}

func TestSaveSecret_Upsert(t *testing.T) {
	v := newTestVault(t)
	ctx := context.Background()
	r := testSecret(t)

	err := v.SaveSecret(ctx, r)
	require.NoError(t, err)

	updated := testSecret(t)
	updated.ID = r.ID
	updated.Name = "updated"
	err = v.SaveSecret(ctx, updated)
	require.NoError(t, err)

	got, err := v.GetSecret(ctx, r.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated", got.Name)
	assert.Equal(t, updated.Data, got.Data)
	assert.Equal(t, updated.DataNonce, got.DataNonce)
}

func TestGetSecret_NotFound(t *testing.T) {
	v := newTestVault(t)

	_, err := v.GetSecret(context.Background(), uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestListSecrets(t *testing.T) {
	v := newTestVault(t)
	ctx := context.Background()

	err := v.SaveSecret(ctx, testSecret(t))
	require.NoError(t, err)
	err = v.SaveSecret(ctx, testSecret(t))
	require.NoError(t, err)

	secrets, err := v.ListSecrets(ctx)
	require.NoError(t, err)
	assert.Len(t, secrets, 2)
}

func TestListSecrets_Empty(t *testing.T) {
	v := newTestVault(t)

	secrets, err := v.ListSecrets(context.Background())
	require.NoError(t, err)
	assert.Empty(t, secrets)
}

func TestDeleteSecret(t *testing.T) {
	v := newTestVault(t)
	ctx := context.Background()
	r := testSecret(t)

	err := v.SaveSecret(ctx, r)
	require.NoError(t, err)

	err = v.DeleteSecret(ctx, r.ID)
	require.NoError(t, err)

	_, err = v.GetSecret(ctx, r.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestMarkSyncedAndListUnsynced(t *testing.T) {
	v := newTestVault(t)
	ctx := context.Background()

	r1 := testSecret(t)
	r1.Synced = false
	r2 := testSecret(t)
	r2.Synced = false

	err := v.SaveSecret(ctx, r1)
	require.NoError(t, err)
	err = v.SaveSecret(ctx, r2)
	require.NoError(t, err)

	unsynced, err := v.ListUnsynced(ctx)
	require.NoError(t, err)
	assert.Len(t, unsynced, 2)

	err = v.MarkSynced(ctx, r1.ID)
	require.NoError(t, err)

	unsynced, err = v.ListUnsynced(ctx)
	require.NoError(t, err)
	require.Len(t, unsynced, 1)
	assert.Equal(t, r2.ID, unsynced[0])
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
