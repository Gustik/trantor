// Package grpcclient содержит gRPC-клиент для взаимодействия с сервером Trantor.
package grpcclient

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Gustik/trantor/api/gen/trantor/v1"
	"github.com/Gustik/trantor/internal/config"
	"github.com/Gustik/trantor/internal/domain"
)

// Client реализует gRPC-соединение с сервером Trantor.
type Client struct {
	conn   *grpc.ClientConn
	auth   pb.AuthServiceClient
	secret pb.SecretServiceClient
}

// New создаёт новое gRPC-соединение с сервером по указанному адресу.
func New(cfg config.ClientConfig) (*Client, error) {
	var creds credentials.TransportCredentials

	if cfg.TLSEnabled {
		creds = credentials.NewClientTLSFromCert(nil, "")
	} else {
		creds = insecure.NewCredentials()
	}

	conn, err := grpc.NewClient(cfg.ServerAddr, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("create grpc connection to %s: %w", cfg.ServerAddr, err)
	}

	return &Client{
		conn:   conn,
		auth:   pb.NewAuthServiceClient(conn),
		secret: pb.NewSecretServiceClient(conn),
	}, nil
}

// Close закрывает gRPC-соединение.
func (c *Client) Close() error {
	return c.conn.Close()
}

// Register регистрирует нового пользователя на сервере.
func (c *Client) Register(ctx context.Context, login string, authKey, encryptedMasterKey, masterKeyNonce, argon2Salt []byte) (token string, err error) {
	r, err := c.auth.Register(ctx, &pb.RegisterRequest{
		Login:              &login,
		AuthKey:            authKey,
		EncryptedMasterKey: encryptedMasterKey,
		MasterKeyNonce:     masterKeyNonce,
		Argon2Salt:         argon2Salt,
	})
	if err != nil {
		return "", toAuthError(err)
	}
	return *r.Token, nil
}

// GetSalt возвращает argon2 salt пользователя с сервера.
func (c *Client) GetSalt(ctx context.Context, login string) ([]byte, error) {
	r, err := c.auth.GetSalt(ctx, &pb.GetSaltRequest{
		Login: &login,
	})
	if err != nil {
		return nil, toAuthError(err)
	}
	return r.Argon2Salt, nil
}

// Login аутентифицирует пользователя и возвращает токен и зашифрованный мастер-ключ.
func (c *Client) Login(ctx context.Context, login string, authKey []byte) (token string, encryptedMasterKey, masterKeyNonce []byte, err error) {
	r, err := c.auth.Login(ctx, &pb.LoginRequest{
		Login:   &login,
		AuthKey: authKey,
	})
	if err != nil {
		return "", nil, nil, toAuthError(err)
	}
	return *r.Token, r.EncryptedMasterKey, r.MasterKeyNonce, nil
}

// CreateSecret отправляет зашифрованный секрет на сервер.
func (c *Client) CreateSecret(ctx context.Context, token string, data, nonce []byte) (id string, err error) {
	r, err := c.secret.CreateSecret(withToken(ctx, token), &pb.CreateSecretRequest{
		Data:  data,
		Nonce: nonce,
	})
	if err != nil {
		return "", toSecretError(err)
	}
	return *r.Id, nil
}

// GetSecret запрашивает секрет с сервера по ID.
func (c *Client) GetSecret(ctx context.Context, token, id string) (*domain.Secret, error) {
	r, err := c.secret.GetSecret(withToken(ctx, token), &pb.GetSecretRequest{
		Id: &id,
	})
	if err != nil {
		return nil, toSecretError(err)
	}
	return protoToSecret(r.GetSecret())
}

// ListSecrets запрашивает список секретов с сервера изменённых после updatedAfter.
func (c *Client) ListSecrets(ctx context.Context, token string, updatedAfter time.Time) ([]*domain.Secret, error) {
	var updatedAfterProto *timestamppb.Timestamp
	if !updatedAfter.IsZero() {
		updatedAfterProto = timestamppb.New(updatedAfter)
	}

	r, err := c.secret.ListSecrets(withToken(ctx, token), &pb.ListSecretsRequest{
		UpdatedAfter: updatedAfterProto,
	})
	if err != nil {
		return nil, toSecretError(err)
	}

	secrets := make([]*domain.Secret, 0, len(r.GetSecrets()))
	for _, s := range r.GetSecrets() {
		secret, err := protoToSecret(s)
		if err != nil {
			return nil, err
		}
		secrets = append(secrets, secret)
	}
	return secrets, nil
}

// UpdateSecret обновляет секрет на сервере.
func (c *Client) UpdateSecret(ctx context.Context, token, id string, data, nonce []byte) error {
	_, err := c.secret.UpdateSecret(withToken(ctx, token), &pb.UpdateSecretRequest{
		Id:    &id,
		Data:  data,
		Nonce: nonce,
	})
	if err != nil {
		return toSecretError(err)
	}
	return nil
}

// DeleteSecret удаляет секрет на сервере.
func (c *Client) DeleteSecret(ctx context.Context, token, id string) error {
	_, err := c.secret.DeleteSecret(withToken(ctx, token), &pb.DeleteSecretRequest{
		Id: &id,
	})
	if err != nil {
		return toSecretError(err)
	}
	return nil
}

// withToken добавляет JWT-токен в исходящие метаданные gRPC запроса.
func withToken(ctx context.Context, token string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
}

// protoToSecret конвертирует proto-сообщение в domain.Secret.
func protoToSecret(s *pb.Secret) (*domain.Secret, error) {
	id, err := uuid.Parse(s.GetId())
	if err != nil {
		return nil, domain.ErrInternal
	}
	return &domain.Secret{
		ID:        id,
		Data:      s.GetData(),
		Nonce:     s.GetNonce(),
		CreatedAt: s.GetCreatedAt().AsTime(),
		UpdatedAt: s.GetUpdatedAt().AsTime(),
	}, nil
}

// TODO: походу каст ошибок надо вынести на уровень сервиса?
//	Должен ли сервис знать коды ошибок protobuf?

// toAuthError транслирует gRPC-ошибку auth-методов в доменную.
func toAuthError(err error) error {
	switch status.Code(err) {
	case codes.NotFound:
		return domain.ErrUserNotFound
	case codes.AlreadyExists:
		return domain.ErrUserAlreadyExists
	case codes.Unauthenticated:
		return domain.ErrInvalidCredentials
	default:
		return domain.ErrInternal
	}
}

// toSecretError транслирует gRPC-ошибку secret-методов в доменную.
func toSecretError(err error) error {
	switch status.Code(err) {
	case codes.NotFound:
		return domain.ErrSecretNotFound
	case codes.Unauthenticated:
		return domain.ErrInvalidCredentials
	default:
		return domain.ErrInternal
	}
}
