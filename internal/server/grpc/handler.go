// Package grpc содержит gRPC-обработчики сервера Trantor.
package grpc

import (
	"context"
	"time"

	"github.com/google/uuid"

	v1 "github.com/Gustik/trantor/api/gen/trantor/v1"
	domain "github.com/Gustik/trantor/internal/server/domain"
)

// authService определяет методы сервиса аутентификации необходимые gRPC-обработчику.
type authService interface {
	// Register регистрирует нового пользователя.
	Register(ctx context.Context, user *domain.User) error
	// GetSalt возвращает argon2 salt пользователя по логину.
	GetSalt(ctx context.Context, login string) ([]byte, error)
	// Login аутентифицирует пользователя и возвращает его данные.
	Login(ctx context.Context, login string, authKey []byte) (*domain.User, error)
}

// secretService определяет методы сервиса секретов необходимые gRPC-обработчику.
type secretService interface {
	// Create создаёт новый секрет.
	Create(ctx context.Context, secret *domain.Secret) error
	// GetByID возвращает секрет по ID.
	GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Secret, error)
	// List возвращает список секретов пользователя.
	List(ctx context.Context, userID uuid.UUID, updatedAfter time.Time) ([]*domain.Secret, error)
	// Update обновляет существующий секрет.
	Update(ctx context.Context, secret *domain.Secret) error
	// Delete удаляет секрет по ID.
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

// Handler реализует gRPC-обработчики сервера Trantor.
type Handler struct {
	v1.UnimplementedAuthServiceServer
	v1.UnimplementedSecretServiceServer
	auth      authService
	secret    secretService
	jwtSecret []byte
}

// New создаёт новый экземпляр Handler.
func New(auth authService, secret secretService, jwtSecret []byte) *Handler {
	return &Handler{auth: auth, secret: secret, jwtSecret: jwtSecret}
}
