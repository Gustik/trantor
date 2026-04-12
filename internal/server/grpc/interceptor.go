package grpc

import (
	"context"
	"strings"

	"github.com/Gustik/trantor/pkg/jwt"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// userIDKey — ключ для хранения userID в контексте запроса.
type userIDKey struct{}

// AuthInterceptor возвращает gRPC UnaryServerInterceptor для проверки JWT-токена.
// Публичные методы (Register, GetSalt, Login) пропускаются без проверки.
func AuthInterceptor(jwtSecret []byte) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// пропускаем публичные методы
		switch info.FullMethod {
		case "/trantor.v1.AuthService/Register",
			"/trantor.v1.AuthService/GetSalt",
			"/trantor.v1.AuthService/Login":
			return handler(ctx, req)
		}

		// для остальных — проверяем токен
		token := extractTokenFromMetadata(ctx)
		userID, err := jwt.ValidateToken(token, jwtSecret)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		ctx = context.WithValue(ctx, userIDKey{}, userID)
		return handler(ctx, req)
	}
}

// UserIDFromContext извлекает userID из контекста запроса.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey{}).(uuid.UUID)
	return id, ok
}

// extractTokenFromMetadata извлекает JWT-токен из метаданных gRPC запроса.
// Ожидает заголовок "authorization" в формате "Bearer <token>".
func extractTokenFromMetadata(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get("authorization")
	if len(values) == 0 {
		return ""
	}
	// обрезаем префикс "Bearer " и возвращаем сам токен
	return strings.TrimPrefix(values[0], "Bearer ")
}
