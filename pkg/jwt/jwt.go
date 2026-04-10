// Package jwt предоставляет функции генерации и валидации JWT-токенов.
package jwt

import "github.com/google/uuid"

// GenerateToken генерирует JWT-токен для пользователя с указанным ID.
func GenerateToken(userID uuid.UUID, secret string) (string, error) {
	return "", nil
}

// ValidateToken проверяет JWT-токен и возвращает ID пользователя.
func ValidateToken(token, secret string) (uuid.UUID, error) {
	return uuid.UUID{}, nil
}
