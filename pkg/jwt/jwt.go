// Package jwt предоставляет функции генерации и валидации JWT-токенов.
package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenTTL — время жизни JWT-токена.
const TokenTTL = 24 * time.Hour

// GenerateToken генерирует JWT-токен для пользователя с указанным ID.
func GenerateToken(userID uuid.UUID, secret []byte) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID.String(),
		"exp": time.Now().Add(TokenTTL).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return signed, nil
}

// ValidateToken проверяет JWT-токен и возвращает ID пользователя.
func ValidateToken(tokenStr string, secret []byte) (uuid.UUID, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return uuid.UUID{}, fmt.Errorf("invalid token")
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return uuid.UUID{}, fmt.Errorf("invalid sub claim")
	}

	id, err := uuid.Parse(sub)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("parse user id: %w", err)
	}

	return id, nil
}
