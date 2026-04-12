package secret

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Gustik/trantor/internal/domain"
	pgstore "github.com/Gustik/trantor/internal/storage/postgres"
)

// mockSecretStorage — мок хранилища секретов.
type mockSecretStorage struct {
	mock.Mock
}

func (m *mockSecretStorage) CreateSecret(ctx context.Context, secret *domain.Secret) error {
	return m.Called(ctx, secret).Error(0)
}

func (m *mockSecretStorage) GetSecretByID(ctx context.Context, id, userID uuid.UUID) (*domain.Secret, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Secret), args.Error(1)
}

func (m *mockSecretStorage) ListSecrets(ctx context.Context, userID uuid.UUID, updatedAfter time.Time) ([]*domain.Secret, error) {
	args := m.Called(ctx, userID, updatedAfter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Secret), args.Error(1)
}

func (m *mockSecretStorage) UpdateSecret(ctx context.Context, secret *domain.Secret) error {
	return m.Called(ctx, secret).Error(0)
}

func (m *mockSecretStorage) DeleteSecret(ctx context.Context, id, userID uuid.UUID) error {
	return m.Called(ctx, id, userID).Error(0)
}

func newTestSecret() *domain.Secret {
	return &domain.Secret{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Data:   []byte("encrypted-data"),
		Nonce:  []byte("nonce-12-bytes!!"),
	}
}

func TestService_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("успешное создание", func(t *testing.T) {
		storage := &mockSecretStorage{}
		storage.On("CreateSecret", ctx, mock.AnythingOfType("*domain.Secret")).Return(nil)

		svc := New(storage)
		secret := newTestSecret()
		err := svc.Create(ctx, secret)
		require.NoError(t, err)

		// сервис должен проставить временные метки
		assert.False(t, secret.CreatedAt.IsZero())
		assert.False(t, secret.UpdatedAt.IsZero())
		assert.Equal(t, secret.CreatedAt, secret.UpdatedAt)
		storage.AssertExpectations(t)
	})

	t.Run("ошибка хранилища", func(t *testing.T) {
		storage := &mockSecretStorage{}
		storage.On("CreateSecret", ctx, mock.AnythingOfType("*domain.Secret")).Return(assert.AnError)

		svc := New(storage)
		err := svc.Create(ctx, newTestSecret())
		assert.ErrorIs(t, err, domain.ErrInternal)
		storage.AssertExpectations(t)
	})
}

func TestService_GetByID(t *testing.T) {
	ctx := context.Background()

	t.Run("найден", func(t *testing.T) {
		secret := newTestSecret()
		storage := &mockSecretStorage{}
		storage.On("GetSecretByID", ctx, secret.ID, secret.UserID).Return(secret, nil)

		svc := New(storage)
		got, err := svc.GetByID(ctx, secret.ID, secret.UserID)
		require.NoError(t, err)
		assert.Equal(t, secret.ID, got.ID)
		storage.AssertExpectations(t)
	})

	t.Run("не найден", func(t *testing.T) {
		id, userID := uuid.New(), uuid.New()
		storage := &mockSecretStorage{}
		storage.On("GetSecretByID", ctx, id, userID).Return(nil, pgstore.ErrNotFound)

		svc := New(storage)
		_, err := svc.GetByID(ctx, id, userID)
		assert.ErrorIs(t, err, domain.ErrSecretNotFound)
		storage.AssertExpectations(t)
	})
}

func TestService_List(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	t.Run("возвращает список", func(t *testing.T) {
		secrets := []*domain.Secret{newTestSecret(), newTestSecret()}
		storage := &mockSecretStorage{}
		storage.On("ListSecrets", ctx, userID, time.Time{}).Return(secrets, nil)

		svc := New(storage)
		got, err := svc.List(ctx, userID, time.Time{})
		require.NoError(t, err)
		assert.Len(t, got, 2)
		storage.AssertExpectations(t)
	})

	t.Run("ошибка хранилища", func(t *testing.T) {
		storage := &mockSecretStorage{}
		storage.On("ListSecrets", ctx, userID, time.Time{}).Return(nil, assert.AnError)

		svc := New(storage)
		_, err := svc.List(ctx, userID, time.Time{})
		assert.ErrorIs(t, err, domain.ErrInternal)
		storage.AssertExpectations(t)
	})
}

func TestService_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("успешное обновление", func(t *testing.T) {
		secret := newTestSecret()
		storage := &mockSecretStorage{}
		storage.On("UpdateSecret", ctx, secret).Return(nil)

		svc := New(storage)
		err := svc.Update(ctx, secret)
		require.NoError(t, err)
		assert.False(t, secret.UpdatedAt.IsZero())
		storage.AssertExpectations(t)
	})

	t.Run("не найден", func(t *testing.T) {
		secret := newTestSecret()
		storage := &mockSecretStorage{}
		storage.On("UpdateSecret", ctx, secret).Return(pgstore.ErrNotFound)

		svc := New(storage)
		err := svc.Update(ctx, secret)
		assert.ErrorIs(t, err, domain.ErrSecretNotFound)
		storage.AssertExpectations(t)
	})

	t.Run("ошибка хранилища", func(t *testing.T) {
		secret := newTestSecret()
		storage := &mockSecretStorage{}
		storage.On("UpdateSecret", ctx, secret).Return(assert.AnError)

		svc := New(storage)
		err := svc.Update(ctx, secret)
		assert.ErrorIs(t, err, domain.ErrInternal)
		storage.AssertExpectations(t)
	})
}

func TestService_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("успешное удаление", func(t *testing.T) {
		id, userID := uuid.New(), uuid.New()
		storage := &mockSecretStorage{}
		storage.On("DeleteSecret", ctx, id, userID).Return(nil)

		svc := New(storage)
		err := svc.Delete(ctx, id, userID)
		require.NoError(t, err)
		storage.AssertExpectations(t)
	})

	t.Run("не найден", func(t *testing.T) {
		id, userID := uuid.New(), uuid.New()
		storage := &mockSecretStorage{}
		storage.On("DeleteSecret", ctx, id, userID).Return(pgstore.ErrNotFound)

		svc := New(storage)
		err := svc.Delete(ctx, id, userID)
		assert.ErrorIs(t, err, domain.ErrSecretNotFound)
		storage.AssertExpectations(t)
	})
}
