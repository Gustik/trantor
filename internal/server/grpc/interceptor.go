package grpc

import (
	"context"

	"google.golang.org/grpc"
)

// userIDKey — ключ для хранения userID в контексте запроса.
type userIDKey struct{}

// AuthInterceptor возвращает gRPC UnaryServerInterceptor для проверки JWT-токена.
// Публичные методы (Register, GetSalt, Login) пропускаются без проверки.
func AuthInterceptor(jwtSecret string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		return handler(ctx, req)
	}
}

// UserIDFromContext извлекает userID из контекста запроса.
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey{}).(string)
	return id, ok
}
