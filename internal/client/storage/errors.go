package storage

import "errors"

// Ошибки хранилища — возвращаются из методов Storage.
// Сервисный слой отвечает за трансляцию этих ошибок в доменные.
var (
	// ErrNotFound возвращается когда запись не найдена в БД.
	ErrNotFound = errors.New("storage: not found")
)
