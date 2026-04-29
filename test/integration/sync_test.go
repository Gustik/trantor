//go:build integration

package integration

import (
	"context"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	clientauth "github.com/Gustik/trantor/internal/client/auth"
	clientdomain "github.com/Gustik/trantor/internal/client/domain"
	grpcclient "github.com/Gustik/trantor/internal/client/grpcclient"
	clientsecret "github.com/Gustik/trantor/internal/client/secret"
	"github.com/Gustik/trantor/internal/client/storage"
	"github.com/Gustik/trantor/internal/common/config"
)

// syncClient — полный стек клиента для интеграционных тестов синхронизации.
type syncClient struct {
	grpc      *grpcclient.Client
	vault     *storage.Vault
	authSvc   *clientauth.Service
	masterKey []byte
}

func newSyncClient(t *testing.T) *syncClient {
	t.Helper()

	vault, err := storage.New(filepath.Join(t.TempDir(), "vault.db"))
	require.NoError(t, err)
	t.Cleanup(func() { vault.Close() })

	gc, err := grpcclient.New(config.ClientConfig{
		ServerAddr: testServerAddr,
		TLSEnabled: false,
	})
	require.NoError(t, err)
	t.Cleanup(func() { gc.Close() })

	return &syncClient{
		grpc:    gc,
		vault:   vault,
		authSvc: clientauth.New(gc, vault),
	}
}

func (c *syncClient) register(t *testing.T, ctx context.Context, login, password string) {
	t.Helper()
	mk, err := c.authSvc.Register(ctx, login, password)
	require.NoError(t, err)
	c.masterKey = mk
}

func (c *syncClient) login(t *testing.T, ctx context.Context, login, password string) {
	t.Helper()
	mk, err := c.authSvc.Login(ctx, login, password)
	require.NoError(t, err)
	c.masterKey = mk
}

func (c *syncClient) svc() *clientsecret.Service {
	return clientsecret.New(c.grpc, c.vault, c.masterKey)
}

// TestSync_BasicPull: клиент A создаёт секрет, клиент B синхронизируется и видит его.
func TestSync_BasicPull(t *testing.T) {
	ctx := context.Background()
	login, password := "sync_basic_user", "hunter2"

	clientA := newSyncClient(t)
	clientA.register(t, ctx, login, password)

	err := clientA.svc().Create(ctx, &clientdomain.SecretPayload{
		Type: clientdomain.SecretTypeText,
		Name: "github-token",
		Data: []byte("ghp_secret"),
	})
	require.NoError(t, err)

	clientB := newSyncClient(t)
	clientB.login(t, ctx, login, password)

	require.NoError(t, clientB.svc().Sync(ctx))

	secrets, err := clientB.vault.ListSecrets(ctx)
	require.NoError(t, err)
	require.Len(t, secrets, 1)
	assert.Equal(t, "github-token", secrets[0].Name)
	assert.Equal(t, clientdomain.SecretTypeText, secrets[0].Type)
}

// TestSync_MultiClient: клиенты A и B независимо создают секреты,
// клиент C синхронизируется и видит оба.
func TestSync_MultiClient(t *testing.T) {
	ctx := context.Background()
	login, password := "sync_multi_user", "hunter2"

	clientA := newSyncClient(t)
	clientA.register(t, ctx, login, password)
	require.NoError(t, clientA.svc().Create(ctx, &clientdomain.SecretPayload{
		Type: clientdomain.SecretTypeLoginPassword,
		Name: "site-a",
		Data: []byte("pass-a"),
	}))

	clientB := newSyncClient(t)
	clientB.login(t, ctx, login, password)
	require.NoError(t, clientB.svc().Create(ctx, &clientdomain.SecretPayload{
		Type: clientdomain.SecretTypeLoginPassword,
		Name: "site-b",
		Data: []byte("pass-b"),
	}))

	clientC := newSyncClient(t)
	clientC.login(t, ctx, login, password)
	require.NoError(t, clientC.svc().Sync(ctx))

	secrets, err := clientC.vault.ListSecrets(ctx)
	require.NoError(t, err)
	require.Len(t, secrets, 2)

	names := []string{secrets[0].Name, secrets[1].Name}
	assert.Contains(t, names, "site-a")
	assert.Contains(t, names, "site-b")
}

// TestSync_DeletePropagation: клиент B удаляет секрет, клиент A синхронизируется
// и перестаёт его видеть.
func TestSync_DeletePropagation(t *testing.T) {
	ctx := context.Background()
	login, password := "sync_delete_user", "hunter2"

	// A регистрируется и создаёт секрет.
	clientA := newSyncClient(t)
	clientA.register(t, ctx, login, password)
	require.NoError(t, clientA.svc().Create(ctx, &clientdomain.SecretPayload{
		Type: clientdomain.SecretTypeText,
		Name: "to-be-deleted",
		Data: []byte("secret-data"),
	}))

	// Первый Sync у A — устанавливает lastSyncedAt.
	require.NoError(t, clientA.svc().Sync(ctx))

	secretsA, err := clientA.vault.ListSecrets(ctx)
	require.NoError(t, err)
	require.Len(t, secretsA, 1)
	secretID := secretsA[0].ID

	// B логинится, синхронизируется и видит секрет.
	clientB := newSyncClient(t)
	clientB.login(t, ctx, login, password)
	require.NoError(t, clientB.svc().Sync(ctx))

	secretsB, err := clientB.vault.ListSecrets(ctx)
	require.NoError(t, err)
	require.Len(t, secretsB, 1)

	// B удаляет секрет с сервера.
	require.NoError(t, clientB.svc().Delete(ctx, secretID))

	secretsB, err = clientB.vault.ListSecrets(ctx)
	require.NoError(t, err)
	assert.Empty(t, secretsB)

	// A синхронизируется и подтягивает удаление.
	require.NoError(t, clientA.svc().Sync(ctx))

	secretsA, err = clientA.vault.ListSecrets(ctx)
	require.NoError(t, err)
	assert.Empty(t, secretsA)
}

// TestSync_DoubleDelete: два клиента удаляют один и тот же секрет.
// Первый получает OK, второй — ErrSecretNotFound.
// После Sync у второго секрет исчезает из локального vault.
func TestSync_DoubleDelete(t *testing.T) {
	ctx := context.Background()
	login, password := "sync_double_delete_user", "hunter2"

	clientA := newSyncClient(t)
	clientA.register(t, ctx, login, password)
	require.NoError(t, clientA.svc().Create(ctx, &clientdomain.SecretPayload{
		Type: clientdomain.SecretTypeText,
		Name: "contested",
		Data: []byte("data"),
	}))
	require.NoError(t, clientA.svc().Sync(ctx))

	secretsA, err := clientA.vault.ListSecrets(ctx)
	require.NoError(t, err)
	require.Len(t, secretsA, 1)
	secretID := secretsA[0].ID

	// B синхронизируется и тоже видит секрет.
	clientB := newSyncClient(t)
	clientB.login(t, ctx, login, password)
	require.NoError(t, clientB.svc().Sync(ctx))

	secretsB, err := clientB.vault.ListSecrets(ctx)
	require.NoError(t, err)
	require.Len(t, secretsB, 1)

	// A удаляет первым — успех.
	require.NoError(t, clientA.svc().Delete(ctx, secretID))

	// B пытается удалить тот же секрет — сервер возвращает ErrSecretNotFound.
	err = clientB.svc().Delete(ctx, secretID)
	assert.ErrorIs(t, err, clientdomain.ErrSecretNotFound)

	// Секрет у B всё ещё в локальном vault (ошибка до vault.DeleteSecret).
	secretsB, err = clientB.vault.ListSecrets(ctx)
	require.NoError(t, err)
	assert.Len(t, secretsB, 1)

	// После Sync B подтягивает мягкое удаление и секрет исчезает.
	require.NoError(t, clientB.svc().Sync(ctx))

	secretsB, err = clientB.vault.ListSecrets(ctx)
	require.NoError(t, err)
	assert.Empty(t, secretsB)
}
