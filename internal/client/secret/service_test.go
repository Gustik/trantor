package secret

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Gustik/trantor/internal/client/domain"
	"github.com/Gustik/trantor/internal/client/storage"
	sdomain "github.com/Gustik/trantor/internal/server/domain"
	"github.com/Gustik/trantor/pkg/crypto"
)

// testMasterKey — фиксированный ключ для тестов (32 байта).
var testMasterKey = bytes.Repeat([]byte{0x42}, 32)

// --- mocks ---

type mockGRPCClient struct{ mock.Mock }

func (m *mockGRPCClient) CreateSecret(ctx context.Context, token string, id uuid.UUID, data, nonce []byte) error {
	return m.Called(ctx, token, id, data, nonce).Error(0)
}

func (m *mockGRPCClient) GetSecret(ctx context.Context, token, id string) (*sdomain.Secret, error) {
	args := m.Called(ctx, token, id)
	s, _ := args.Get(0).(*sdomain.Secret)
	return s, args.Error(1)
}

func (m *mockGRPCClient) ListSecrets(ctx context.Context, token string, updatedAfter time.Time) ([]*sdomain.Secret, error) {
	args := m.Called(ctx, token, updatedAfter)
	ss, _ := args.Get(0).([]*sdomain.Secret)
	return ss, args.Error(1)
}

func (m *mockGRPCClient) UpdateSecret(ctx context.Context, token, id string, data, nonce []byte) error {
	return m.Called(ctx, token, id, data, nonce).Error(0)
}

func (m *mockGRPCClient) DeleteSecret(ctx context.Context, token, id string) error {
	return m.Called(ctx, token, id).Error(0)
}

type mockVault struct{ mock.Mock }

func (m *mockVault) SaveSecret(ctx context.Context, r *domain.Secret) error {
	return m.Called(ctx, r).Error(0)
}

func (m *mockVault) MarkSynced(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockVault) ListUnsynced(ctx context.Context) ([]uuid.UUID, error) {
	args := m.Called(ctx)
	ids, _ := args.Get(0).([]uuid.UUID)
	return ids, args.Error(1)
}

func (m *mockVault) GetSecret(ctx context.Context, id uuid.UUID) (*domain.Secret, error) {
	args := m.Called(ctx, id)
	s, _ := args.Get(0).(*domain.Secret)
	return s, args.Error(1)
}

func (m *mockVault) ListSecrets(ctx context.Context) ([]*domain.Secret, error) {
	args := m.Called(ctx)
	ss, _ := args.Get(0).([]*domain.Secret)
	return ss, args.Error(1)
}

func (m *mockVault) DeleteSecret(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockVault) LastSyncedAt(ctx context.Context) (time.Time, error) {
	args := m.Called(ctx)
	return args.Get(0).(time.Time), args.Error(1)
}

func (m *mockVault) SetLastSyncedAt(ctx context.Context, t time.Time) error {
	return m.Called(ctx, t).Error(0)
}

func (m *mockVault) GetAuthToken(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

// --- helpers ---

func testPayload() *domain.SecretPayload {
	return &domain.SecretPayload{
		Type: domain.SecretTypeText,
		Name: "mysite.com",
		Data: []byte("hunter2"),
	}
}

// encryptedVaultSecret возвращает domain.Secret с Data зашифрованной testMasterKey.
func encryptedVaultSecret(t *testing.T, payload *domain.SecretPayload) *domain.Secret {
	t.Helper()
	data, nonce, err := crypto.Encrypt(testMasterKey, payload.Data)
	require.NoError(t, err)
	return &domain.Secret{
		ID:        uuid.New(),
		Type:      payload.Type,
		Name:      payload.Name,
		Data:      data,
		DataNonce: nonce,
		Metadata:  payload.Metadata,
		UpdatedAt: time.Now().UTC(),
		Synced:    true,
	}
}

// encryptedServerSecret возвращает sdomain.Secret с полным payload зашифрованным testMasterKey.
func encryptedServerSecret(t *testing.T, payload *domain.SecretPayload) *sdomain.Secret {
	t.Helper()
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	data, nonce, err := crypto.Encrypt(testMasterKey, raw)
	require.NoError(t, err)
	return &sdomain.Secret{
		ID:        uuid.New(),
		Data:      data,
		Nonce:     nonce,
		UpdatedAt: time.Now().UTC(),
	}
}

// --- Create ---

func TestCreate(t *testing.T) {
	ctx := context.Background()
	payload := testPayload()

	t.Run("успешное создание с синхронизацией", func(t *testing.T) {
		c := &mockGRPCClient{}
		v := &mockVault{}
		anyBytes := mock.AnythingOfType("[]uint8")

		v.On("SaveSecret", ctx, mock.MatchedBy(func(s *domain.Secret) bool {
			// данные должны быть зашифрованы, не совпадать с plaintext
			return !bytes.Equal(s.Data, payload.Data) && s.Name == payload.Name && !s.Synced
		})).Return(nil)
		v.On("GetAuthToken", ctx).Return("token", nil)
		c.On("CreateSecret", ctx, "token", mock.AnythingOfType("uuid.UUID"), anyBytes, anyBytes).Return(nil)
		v.On("MarkSynced", ctx, mock.AnythingOfType("uuid.UUID")).Return(nil)

		err := New(c, v, testMasterKey).Create(ctx, payload)
		require.NoError(t, err)
		c.AssertExpectations(t)
		v.AssertExpectations(t)
	})

	t.Run("сервер недоступен — сохраняется локально без ошибки", func(t *testing.T) {
		c := &mockGRPCClient{}
		v := &mockVault{}
		anyBytes := mock.AnythingOfType("[]uint8")

		v.On("SaveSecret", ctx, mock.Anything).Return(nil)
		v.On("GetAuthToken", ctx).Return("token", nil)
		c.On("CreateSecret", ctx, "token", mock.AnythingOfType("uuid.UUID"), anyBytes, anyBytes).Return(assert.AnError)

		err := New(c, v, testMasterKey).Create(ctx, payload)
		assert.NoError(t, err) // offline-first: ошибка сервера не возвращается
	})

	t.Run("нет токена", func(t *testing.T) {
		c := &mockGRPCClient{}
		v := &mockVault{}

		v.On("SaveSecret", ctx, mock.Anything).Return(nil)
		v.On("GetAuthToken", ctx).Return("", storage.ErrNotFound)

		err := New(c, v, testMasterKey).Create(ctx, payload)
		assert.ErrorIs(t, err, domain.ErrNotAuthenticated)
	})
}

// --- Get ---

func TestGet(t *testing.T) {
	ctx := context.Background()
	payload := testPayload()
	vaultSecret := encryptedVaultSecret(t, payload)

	t.Run("успешно", func(t *testing.T) {
		v := &mockVault{}
		v.On("GetSecret", ctx, vaultSecret.ID).Return(vaultSecret, nil)

		got, err := New(nil, v, testMasterKey).Get(ctx, vaultSecret.ID)
		require.NoError(t, err)
		assert.Equal(t, payload.Name, got.Name)
		assert.Equal(t, payload.Data, got.Data)
		assert.Equal(t, payload.Type, got.Type)
	})

	t.Run("не найден", func(t *testing.T) {
		v := &mockVault{}
		id := uuid.New()
		v.On("GetSecret", ctx, id).Return(nil, storage.ErrNotFound)

		_, err := New(nil, v, testMasterKey).Get(ctx, id)
		assert.ErrorIs(t, err, domain.ErrSecretNotFound)
	})
}

// --- List ---

func TestList(t *testing.T) {
	ctx := context.Background()

	t.Run("пустой список", func(t *testing.T) {
		v := &mockVault{}
		v.On("ListSecrets", ctx).Return([]*domain.Secret{}, nil)

		got, err := New(nil, v, testMasterKey).List(ctx)
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("несколько секретов", func(t *testing.T) {
		p1 := testPayload()
		p2 := &domain.SecretPayload{Type: domain.SecretTypeLoginPassword, Name: "bank", Data: []byte("pass")}
		v := &mockVault{}
		v.On("ListSecrets", ctx).Return([]*domain.Secret{
			encryptedVaultSecret(t, p1),
			encryptedVaultSecret(t, p2),
		}, nil)

		got, err := New(nil, v, testMasterKey).List(ctx)
		require.NoError(t, err)
		assert.Len(t, got, 2)
		assert.Equal(t, p1.Name, got[0].Name)
		assert.Equal(t, p2.Name, got[1].Name)
	})

	t.Run("ошибка хранилища", func(t *testing.T) {
		v := &mockVault{}
		v.On("ListSecrets", ctx).Return(nil, assert.AnError)

		_, err := New(nil, v, testMasterKey).List(ctx)
		assert.Error(t, err)
	})
}

// --- Delete ---

func TestDelete(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()

	t.Run("успешное удаление", func(t *testing.T) {
		c := &mockGRPCClient{}
		v := &mockVault{}
		v.On("GetAuthToken", ctx).Return("token", nil)
		c.On("DeleteSecret", ctx, "token", id.String()).Return(nil)
		v.On("DeleteSecret", ctx, id).Return(nil)

		err := New(c, v, nil).Delete(ctx, id)
		require.NoError(t, err)
		c.AssertExpectations(t)
		v.AssertExpectations(t)
	})

	t.Run("нет токена", func(t *testing.T) {
		v := &mockVault{}
		v.On("GetAuthToken", ctx).Return("", storage.ErrNotFound)

		err := New(nil, v, nil).Delete(ctx, id)
		assert.ErrorIs(t, err, domain.ErrNotAuthenticated)
	})

	t.Run("секрет не найден на сервере — локально не трогаем", func(t *testing.T) {
		c := &mockGRPCClient{}
		v := &mockVault{}
		v.On("GetAuthToken", ctx).Return("token", nil)
		c.On("DeleteSecret", ctx, "token", id.String()).Return(domain.ErrSecretNotFound)

		err := New(c, v, nil).Delete(ctx, id)
		assert.Error(t, err)
		v.AssertNotCalled(t, "DeleteSecret", ctx, id) // локальный vault не трогается
	})
}

// --- Sync ---

func TestSync(t *testing.T) {
	ctx := context.Background()

	t.Run("нет токена", func(t *testing.T) {
		v := &mockVault{}
		v.On("GetAuthToken", ctx).Return("", storage.ErrNotFound)

		err := New(nil, v, testMasterKey).Sync(ctx)
		assert.ErrorIs(t, err, domain.ErrNotAuthenticated)
	})

	t.Run("push несинхронизированного секрета", func(t *testing.T) {
		payload := testPayload()
		vaultSecret := encryptedVaultSecret(t, payload)
		vaultSecret.Synced = false

		c := &mockGRPCClient{}
		v := &mockVault{}
		anyBytes := mock.AnythingOfType("[]uint8")

		v.On("GetAuthToken", ctx).Return("token", nil)
		v.On("ListUnsynced", ctx).Return([]uuid.UUID{vaultSecret.ID}, nil)
		v.On("GetSecret", ctx, vaultSecret.ID).Return(vaultSecret, nil)
		c.On("CreateSecret", ctx, "token", vaultSecret.ID, anyBytes, anyBytes).Return(nil)
		v.On("MarkSynced", ctx, vaultSecret.ID).Return(nil)
		v.On("LastSyncedAt", ctx).Return(time.Time{}, nil)
		c.On("ListSecrets", ctx, "token", time.Time{}).Return([]*sdomain.Secret{}, nil)
		v.On("SetLastSyncedAt", ctx, mock.AnythingOfType("time.Time")).Return(nil)

		err := New(c, v, testMasterKey).Sync(ctx)
		require.NoError(t, err)
		c.AssertExpectations(t)
		v.AssertExpectations(t)
	})

	t.Run("pull нового секрета с сервера", func(t *testing.T) {
		payload := testPayload()
		serverSecret := encryptedServerSecret(t, payload)

		c := &mockGRPCClient{}
		v := &mockVault{}

		v.On("GetAuthToken", ctx).Return("token", nil)
		v.On("ListUnsynced", ctx).Return([]uuid.UUID{}, nil)
		v.On("LastSyncedAt", ctx).Return(time.Time{}, nil)
		c.On("ListSecrets", ctx, "token", time.Time{}).Return([]*sdomain.Secret{serverSecret}, nil)
		v.On("SaveSecret", ctx, mock.MatchedBy(func(s *domain.Secret) bool {
			return s.ID == serverSecret.ID && s.Name == payload.Name && s.Synced
		})).Return(nil)
		v.On("SetLastSyncedAt", ctx, mock.AnythingOfType("time.Time")).Return(nil)

		err := New(c, v, testMasterKey).Sync(ctx)
		require.NoError(t, err)
		c.AssertExpectations(t)
		v.AssertExpectations(t)
	})

	t.Run("pull удалённого секрета — удаляем локально", func(t *testing.T) {
		deletedAt := time.Now()
		serverSecret := &sdomain.Secret{
			ID:        uuid.New(),
			DeletedAt: &deletedAt,
			UpdatedAt: deletedAt,
		}

		c := &mockGRPCClient{}
		v := &mockVault{}

		v.On("GetAuthToken", ctx).Return("token", nil)
		v.On("ListUnsynced", ctx).Return([]uuid.UUID{}, nil)
		v.On("LastSyncedAt", ctx).Return(time.Time{}, nil)
		c.On("ListSecrets", ctx, "token", time.Time{}).Return([]*sdomain.Secret{serverSecret}, nil)
		v.On("DeleteSecret", ctx, serverSecret.ID).Return(nil)
		v.On("SetLastSyncedAt", ctx, mock.AnythingOfType("time.Time")).Return(nil)

		err := New(c, v, testMasterKey).Sync(ctx)
		require.NoError(t, err)
		v.AssertCalled(t, "DeleteSecret", ctx, serverSecret.ID)
	})
}
