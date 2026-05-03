// Package domain содержит общие бизнес-ошибки доменного слоя.
package domain

import "errors"

// Ошибки доменного слоя для пользователей.
var (
	// ErrUserNotFound возвращается когда пользователь не найден.
	ErrUserNotFound = errors.New("пользователь не найден")
	// ErrUserAlreadyExists возвращается при попытке зарегистрировать уже существующий логин.
	ErrUserAlreadyExists = errors.New("пользователь уже существует")
	// ErrInvalidCredentials возвращается при неверном auth_key во время входа.
	ErrInvalidCredentials = errors.New("неверные учётные данные")
)
