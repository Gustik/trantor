package domain

import "errors"

var (
	// ErrNotAuthenticated
	ErrNotAuthenticated = errors.New("не авторизован")
)
