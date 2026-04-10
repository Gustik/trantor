// Package config содержит конфигурацию сервера и клиента Trantor.
package config

// ServerConfig содержит конфигурацию сервера Trantor.
type ServerConfig struct {
	// GRPC — адрес на котором сервер принимает входящие соединения, например ":50051".
	GRPC string
	// DSN — строка подключения к PostgreSQL, например "postgres://user:pass@localhost:5432/trantor?sslmode=disable".
	DSN string
	// JWTSecret — секрет для подписи JWT-токенов. Должен быть длинным и случайным.
	JWTSecret string
}

// ClientConfig содержит конфигурацию CLI-клиента Trantor.
type ClientConfig struct {
	// ServerAddr — адрес сервера Trantor, например "localhost:50051".
	ServerAddr string
	// VaultPath — путь к файлу локального хранилища секретов, например "~/.trantor/vault.db".
	VaultPath string
	// JWTSecret — секрет для валидации JWT-токенов.
	JWTSecret string
}

// LoadServer загружает конфигурацию сервера из переменных окружения.
func LoadServer() (*ServerConfig, error) {
	return nil, nil
}

// LoadClient загружает конфигурацию клиента из переменных окружения.
func LoadClient() (*ClientConfig, error) {
	return nil, nil
}
