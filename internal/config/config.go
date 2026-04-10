// Package config содержит конфигурацию сервера и клиента Trantor.
package config

import "github.com/ilyakaznacheev/cleanenv"

// ServerConfig содержит конфигурацию сервера Trantor.
type ServerConfig struct {
	// GRPC — адрес на котором сервер принимает входящие соединения, например ":50051".
	GRPC string `env:"TRANTOR_GRPC" env-default:":50051"`
	// DSN — строка подключения к PostgreSQL.
	DSN string `env:"TRANTOR_DSN" env-required:"true"`
	// JWTSecret — секрет для подписи JWT-токенов. Должен быть длинным и случайным.
	JWTSecret string `env:"TRANTOR_JWT_SECRET" env-required:"true"`
}

// ClientConfig содержит конфигурацию CLI-клиента Trantor.
type ClientConfig struct {
	// ServerAddr — адрес сервера Trantor, например "localhost:50051".
	ServerAddr string `env:"TRANTOR_SERVER_ADDR" env-default:"localhost:50051"`
	// VaultPath — путь к файлу локального хранилища секретов.
	VaultPath string `env:"TRANTOR_VAULT_PATH" env-default:"~/.trantor/vault.db"`
}

// LoadServer загружает конфигурацию сервера из переменных окружения.
func LoadServer() (*ServerConfig, error) {
	cfg := &ServerConfig{}
	if err := cleanenv.ReadEnv(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// LoadClient загружает конфигурацию клиента из переменных окружения.
func LoadClient() (*ClientConfig, error) {
	cfg := &ClientConfig{}
	if err := cleanenv.ReadEnv(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
