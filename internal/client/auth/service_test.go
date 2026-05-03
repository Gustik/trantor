package auth

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Gustik/trantor/internal/client/storage"
	commondomain "github.com/Gustik/trantor/internal/common/domain"
	"github.com/Gustik/trantor/pkg/crypto"
)

// --- mocks ---

type mockGRPCClient struct{ mock.Mock }

func (m *mockGRPCClient) Register(ctx context.Context, login string, authKey, encryptedMasterKey, masterKeyNonce, argon2Salt []byte) (string, error) {
	args := m.Called(ctx, login, authKey, encryptedMasterKey, masterKeyNonce, argon2Salt)
	return args.String(0), args.Error(1)
}

func (m *mockGRPCClient) GetSalt(ctx context.Context, login string) ([]byte, error) {
	args := m.Called(ctx, login)
	b, _ := args.Get(0).([]byte)
	return b, args.Error(1)
}

func (m *mockGRPCClient) Login(ctx context.Context, login string, authKey []byte) (string, []byte, []byte, error) {
	args := m.Called(ctx, login, authKey)
	encMK, _ := args.Get(1).([]byte)
	nonce, _ := args.Get(2).([]byte)
	return args.String(0), encMK, nonce, args.Error(3)
}

type mockVault struct{ mock.Mock }

func (m *mockVault) SetAuthToken(ctx context.Context, token string) error {
	return m.Called(ctx, token).Error(0)
}

func (m *mockVault) GetAuthToken(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *mockVault) SetAuthCache(ctx context.Context, salt, encryptedMasterKey, masterKeyNonce []byte) error {
	return m.Called(ctx, salt, encryptedMasterKey, masterKeyNonce).Error(0)
}

func (m *mockVault) GetAuthCache(ctx context.Context) ([]byte, []byte, []byte, error) {
	args := m.Called(ctx)
	salt, _ := args.Get(0).([]byte)
	encMK, _ := args.Get(1).([]byte)
	nonce, _ := args.Get(2).([]byte)
	return salt, encMK, nonce, args.Error(3)
}

// --- helpers ---

const testPassword = "test-password-123"

var testSalt = bytes.Repeat([]byte{0xAA}, 32)

// cryptoMaterial возвращает фиксированный мастер-ключ и его зашифрованную версию.
// DeriveKeys использует Argon2 (~100ms), поэтому вызывается только там где нужно.
func cryptoMaterial(t *testing.T) (masterKey, encryptedMasterKey, nonce []byte) {
	t.Helper()
	_, encryptionKey := crypto.DeriveKeys(testPassword, testSalt)
	masterKey = bytes.Repeat([]byte{0x42}, 32)
	var err error
	encryptedMasterKey, nonce, err = crypto.Encrypt(encryptionKey, masterKey)
	require.NoError(t, err)
	return
}

// --- Register ---

func TestRegister(t *testing.T) {
	ctx := context.Background()
	anyBytes := mock.AnythingOfType("[]uint8")

	t.Run("успешная регистрация", func(t *testing.T) {
		c := &mockGRPCClient{}
		v := &mockVault{}
		c.On("Register", ctx, "alice", anyBytes, anyBytes, anyBytes, anyBytes).Return("token", nil)
		v.On("SetAuthToken", ctx, "token").Return(nil)
		v.On("SetAuthCache", ctx, anyBytes, anyBytes, anyBytes).Return(nil)

		masterKey, err := New(c, v).Register(ctx, "alice", testPassword)
		require.NoError(t, err)
		assert.Len(t, masterKey, 32)
		c.AssertExpectations(t)
		v.AssertExpectations(t)
	})

	t.Run("ошибка сервера", func(t *testing.T) {
		c := &mockGRPCClient{}
		v := &mockVault{}
		c.On("Register", ctx, "alice", anyBytes, anyBytes, anyBytes, anyBytes).Return("", assert.AnError)

		_, err := New(c, v).Register(ctx, "alice", testPassword)
		assert.ErrorIs(t, err, commondomain.ErrInternal)
	})

	t.Run("ошибка сохранения токена", func(t *testing.T) {
		c := &mockGRPCClient{}
		v := &mockVault{}
		c.On("Register", ctx, "alice", anyBytes, anyBytes, anyBytes, anyBytes).Return("token", nil)
		v.On("SetAuthToken", ctx, "token").Return(assert.AnError)

		_, err := New(c, v).Register(ctx, "alice", testPassword)
		assert.Error(t, err)
	})

	t.Run("ошибка сохранения кэша", func(t *testing.T) {
		c := &mockGRPCClient{}
		v := &mockVault{}
		c.On("Register", ctx, "alice", anyBytes, anyBytes, anyBytes, anyBytes).Return("token", nil)
		v.On("SetAuthToken", ctx, "token").Return(nil)
		v.On("SetAuthCache", ctx, anyBytes, anyBytes, anyBytes).Return(assert.AnError)

		_, err := New(c, v).Register(ctx, "alice", testPassword)
		assert.Error(t, err)
	})
}

// --- Login ---

func TestLogin(t *testing.T) {
	ctx := context.Background()
	wantMasterKey, encMK, nonce := cryptoMaterial(t)
	authKey, _ := crypto.DeriveKeys(testPassword, testSalt)

	t.Run("успешный вход", func(t *testing.T) {
		c := &mockGRPCClient{}
		v := &mockVault{}
		c.On("GetSalt", ctx, "alice").Return(testSalt, nil)
		c.On("Login", ctx, "alice", authKey).Return("token", encMK, nonce, nil)
		v.On("SetAuthToken", ctx, "token").Return(nil)
		v.On("SetAuthCache", ctx, testSalt, encMK, nonce).Return(nil)

		got, err := New(c, v).Login(ctx, "alice", testPassword)
		require.NoError(t, err)
		assert.Equal(t, wantMasterKey, got)
		c.AssertExpectations(t)
		v.AssertExpectations(t)
	})

	t.Run("ошибка получения salt", func(t *testing.T) {
		c := &mockGRPCClient{}
		v := &mockVault{}
		c.On("GetSalt", ctx, "nobody").Return(nil, assert.AnError)

		_, err := New(c, v).Login(ctx, "nobody", testPassword)
		assert.ErrorIs(t, err, commondomain.ErrInternal)
	})

	t.Run("неверные credentials", func(t *testing.T) {
		c := &mockGRPCClient{}
		v := &mockVault{}
		c.On("GetSalt", ctx, "alice").Return(testSalt, nil)
		c.On("Login", ctx, "alice", authKey).Return("", nil, nil, commondomain.ErrInvalidCredentials)

		_, err := New(c, v).Login(ctx, "alice", testPassword)
		assert.ErrorIs(t, err, commondomain.ErrInvalidCredentials)
	})

	t.Run("повреждённый мастер-ключ на сервере", func(t *testing.T) {
		c := &mockGRPCClient{}
		v := &mockVault{}
		garbage := bytes.Repeat([]byte{0xFF}, 32)
		c.On("GetSalt", ctx, "alice").Return(testSalt, nil)
		c.On("Login", ctx, "alice", authKey).Return("token", garbage, nonce, nil)

		_, err := New(c, v).Login(ctx, "alice", testPassword)
		assert.Error(t, err)
	})
}

// --- DeriveFromCache ---

func TestDeriveFromCache(t *testing.T) {
	ctx := context.Background()
	wantMasterKey, encMK, nonce := cryptoMaterial(t)

	t.Run("успешно", func(t *testing.T) {
		v := &mockVault{}
		v.On("GetAuthCache", ctx).Return(testSalt, encMK, nonce, nil)

		got, err := New(nil, v).DeriveFromCache(ctx, testPassword)
		require.NoError(t, err)
		assert.Equal(t, wantMasterKey, got)
	})

	t.Run("неверный пароль", func(t *testing.T) {
		v := &mockVault{}
		v.On("GetAuthCache", ctx).Return(testSalt, encMK, nonce, nil)

		_, err := New(nil, v).DeriveFromCache(ctx, "wrong-password")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "неверный пароль")
	})

	t.Run("кэш не найден", func(t *testing.T) {
		v := &mockVault{}
		v.On("GetAuthCache", ctx).Return(nil, nil, nil, storage.ErrNotFound)

		_, err := New(nil, v).DeriveFromCache(ctx, testPassword)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
}
