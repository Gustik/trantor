// Общие ошибки доменного слоя
package domain

import "errors"

var (
	ErrInternal         = errors.New("internal error")
	ErrNotAuthenticated = errors.New("не авторизован")
)
