// Package config содержит конфигурацию сервера и клиента Trantor.
package config

import (
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// ServerConfig содержит конфигурацию сервера Trantor.
type ServerConfig struct {
	// GRPC — адрес на котором сервер принимает входящие соединения, например ":50051".
	GRPC string `env:"TRANTOR_GRPC" env-default:":50051"`
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

// DBConfig содержит конфигурацию postgres
type DBConfig struct {
	DSN             string        `env:"TRANTOR_DSN" env-required:"true"`
	MaxConns        int           `env:"TRANTOR_DB_MAX_CONNS" env-default:"10"`
	MinConns        int           `env:"TRANTOR_DB_MIN_CONNS" env-default:"2"`
	MaxConnLifetime time.Duration `env:"TRANTOR_DB_MAX_CONN_LIFETIME" env-default:"1h"`
	MaxConnIdleTime time.Duration `env:"TRANTOR_DB_MAX_CONN_IDLE_TIME" env-default:"30m"`
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

// LoadDB загружает конфигурацию postgres из переменных окружения.
func LoadDB() (*DBConfig, error) {
	cfg := &DBConfig{}
	if err := cleanenv.ReadEnv(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
