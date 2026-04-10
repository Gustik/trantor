// Package grpcclient содержит gRPC-клиент для взаимодействия с сервером Trantor.
package grpcclient

import (
	"context"
	"time"

	"github.com/Gustik/trantor/internal/domain"
	"google.golang.org/grpc"
)

// Client реализует gRPC-соединение с сервером Trantor.
type Client struct {
	conn *grpc.ClientConn
}

// New создаёт новое gRPC-соединение с сервером по указанному адресу.
func New(addr string) (*Client, error) {
	return nil, nil
}

// Close закрывает gRPC-соединение.
func (c *Client) Close() error {
	return nil
}

// Register регистрирует нового пользователя на сервере.
func (c *Client) Register(ctx context.Context, login string, authKey, encryptedMasterKey, masterKeyNonce, argon2Salt []byte) (token string, err error) {
	return "", nil
}

// GetSalt возвращает argon2 salt пользователя с сервера.
func (c *Client) GetSalt(ctx context.Context, login string) ([]byte, error) {
	return nil, nil
}

// Login аутентифицирует пользователя и возвращает токен и зашифрованный мастер-ключ.
func (c *Client) Login(ctx context.Context, login string, authKey []byte) (token string, encryptedMasterKey, masterKeyNonce []byte, err error) {
	return "", nil, nil, nil
}

// CreateSecret отправляет зашифрованный секрет на сервер.
func (c *Client) CreateSecret(ctx context.Context, token string, data, nonce []byte) (id string, err error) {
	return "", nil
}

// GetSecret запрашивает секрет с сервера по ID.
func (c *Client) GetSecret(ctx context.Context, token, id string) (*domain.Secret, error) {
	return nil, nil
}

// ListSecrets запрашивает список секретов с сервера изменённых после updatedAfter.
func (c *Client) ListSecrets(ctx context.Context, token string, updatedAfter time.Time) ([]*domain.Secret, error) {
	return nil, nil
}

// UpdateSecret обновляет секрет на сервере.
func (c *Client) UpdateSecret(ctx context.Context, token, id string, data, nonce []byte) error {
	return nil
}

// DeleteSecret удаляет секрет на сервере.
func (c *Client) DeleteSecret(ctx context.Context, token, id string) error {
	return nil
}
