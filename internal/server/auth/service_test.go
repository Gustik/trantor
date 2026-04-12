package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/Gustik/trantor/internal/domain"
	pgstore "github.com/Gustik/trantor/internal/storage/postgres"
)

// mockUserStorage — мок хранилища пользователей.
type mockUserStorage struct {
	mock.Mock
}

func (m *mockUserStorage) CreateUser(ctx context.Context, user *domain.User) error {
	return m.Called(ctx, user).Error(0)
}

func (m *mockUserStorage) FindUserByLogin(ctx context.Context, login string) (*domain.User, error) {
	args := m.Called(ctx, login)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserStorage) FindUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func newTestUser() *domain.User {
	return &domain.User{
		ID:                 uuid.New(),
		Login:              "testuser",
		AuthKeyHash:        "raw-auth-key",
		EncryptedMasterKey: []byte("encrypted"),
		MasterKeyNonce:     []byte("nonce"),
		Argon2Salt:         []byte("salt"),
		CreatedAt:          time.Now(),
	}
}

func TestService_Register(t *testing.T) {
	ctx := context.Background()

	t.Run("успешная регистрация", func(t *testing.T) {
		storage := &mockUserStorage{}
		storage.On("CreateUser", ctx, mock.AnythingOfType("*domain.User")).Return(nil)

		svc := New(storage)
		user := newTestUser()
		rawKey := user.AuthKeyHash

		err := svc.Register(ctx, user)
		require.NoError(t, err)

		// AuthKeyHash должен стать bcrypt-хешем
		assert.NotEqual(t, rawKey, user.AuthKeyHash)
		assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(user.AuthKeyHash), []byte(rawKey)))
		storage.AssertExpectations(t)
	})

	t.Run("логин уже занят", func(t *testing.T) {
		storage := &mockUserStorage{}
		storage.On("CreateUser", ctx, mock.AnythingOfType("*domain.User")).Return(pgstore.ErrDuplicate)

		svc := New(storage)
		err := svc.Register(ctx, newTestUser())
		assert.ErrorIs(t, err, domain.ErrUserAlreadyExists)
		storage.AssertExpectations(t)
	})

	t.Run("ошибка хранилища", func(t *testing.T) {
		storage := &mockUserStorage{}
		storage.On("CreateUser", ctx, mock.AnythingOfType("*domain.User")).Return(assert.AnError)

		svc := New(storage)
		err := svc.Register(ctx, newTestUser())
		assert.ErrorIs(t, err, domain.ErrInternal)
		storage.AssertExpectations(t)
	})
}

func TestService_GetSalt(t *testing.T) {
	ctx := context.Background()

	t.Run("успешно", func(t *testing.T) {
		user := newTestUser()
		storage := &mockUserStorage{}
		storage.On("FindUserByLogin", ctx, user.Login).Return(user, nil)

		svc := New(storage)
		salt, err := svc.GetSalt(ctx, user.Login)
		require.NoError(t, err)
		assert.Equal(t, user.Argon2Salt, salt)
		storage.AssertExpectations(t)
	})

	t.Run("пользователь не найден", func(t *testing.T) {
		storage := &mockUserStorage{}
		storage.On("FindUserByLogin", ctx, "nobody").Return(nil, pgstore.ErrNotFound)

		svc := New(storage)
		_, err := svc.GetSalt(ctx, "nobody")
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		storage.AssertExpectations(t)
	})
}

func TestService_Login(t *testing.T) {
	ctx := context.Background()
	rawKey := []byte("raw-auth-key")

	hash, err := bcrypt.GenerateFromPassword(rawKey, bcrypt.DefaultCost)
	require.NoError(t, err)

	user := newTestUser()
	user.AuthKeyHash = string(hash)

	t.Run("успешный логин", func(t *testing.T) {
		storage := &mockUserStorage{}
		storage.On("FindUserByLogin", ctx, user.Login).Return(user, nil)

		svc := New(storage)
		got, err := svc.Login(ctx, user.Login, rawKey)
		require.NoError(t, err)
		assert.Equal(t, user.ID, got.ID)
		storage.AssertExpectations(t)
	})

	t.Run("неверный auth_key", func(t *testing.T) {
		storage := &mockUserStorage{}
		storage.On("FindUserByLogin", ctx, user.Login).Return(user, nil)

		svc := New(storage)
		_, err := svc.Login(ctx, user.Login, []byte("wrong-key"))
		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
		storage.AssertExpectations(t)
	})

	t.Run("пользователь не найден", func(t *testing.T) {
		storage := &mockUserStorage{}
		storage.On("FindUserByLogin", ctx, "nobody").Return(nil, pgstore.ErrNotFound)

		svc := New(storage)
		_, err := svc.Login(ctx, "nobody", rawKey)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		storage.AssertExpectations(t)
	})
}

func TestService_GetUserByID(t *testing.T) {
	ctx := context.Background()

	t.Run("найден", func(t *testing.T) {
		user := newTestUser()
		storage := &mockUserStorage{}
		storage.On("FindUserByID", ctx, user.ID).Return(user, nil)

		svc := New(storage)
		got, err := svc.GetUserByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, user.ID, got.ID)
		storage.AssertExpectations(t)
	})

	t.Run("не найден", func(t *testing.T) {
		id := uuid.New()
		storage := &mockUserStorage{}
		storage.On("FindUserByID", ctx, id).Return(nil, pgstore.ErrNotFound)

		svc := New(storage)
		_, err := svc.GetUserByID(ctx, id)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		storage.AssertExpectations(t)
	})
}
