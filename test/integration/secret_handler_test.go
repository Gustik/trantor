//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	pb "github.com/Gustik/trantor/api/gen/trantor/v1"
	"github.com/Gustik/trantor/pkg/crypto"
)

// authCtx возвращает контекст с JWT-токеном для авторизованных запросов.
func authCtx(t *testing.T, login string) context.Context {
	t.Helper()
	ctx := context.Background()
	resp, err := testAuthClient.Register(ctx, validRegisterReq(login))
	if err != nil {
		// пользователь уже существует — логинимся
		loginStr := login
		loginResp, err := testAuthClient.Login(ctx, &pb.LoginRequest{
			Login:   &loginStr,
			AuthKey: validAuthKey,
		})
		require.NoError(t, err)
		md := metadata.Pairs("authorization", "Bearer "+loginResp.GetToken())
		return metadata.NewOutgoingContext(ctx, md)
	}
	md := metadata.Pairs("authorization", "Bearer "+resp.GetToken())
	return metadata.NewOutgoingContext(ctx, md)
}

func validSecretReq() *pb.CreateSecretRequest {
	return &pb.CreateSecretRequest{
		Data:  []byte("encrypted-secret-data"),
		Nonce: make([]byte, crypto.NonceSize),
	}
}

func TestHandler_CreateSecret(t *testing.T) {
	ctx := authCtx(t, "grpc_secret_alice")

	t.Run("успешное создание", func(t *testing.T) {
		resp, err := testSecretClient.CreateSecret(ctx, validSecretReq())
		require.NoError(t, err)
		assert.NotEmpty(t, resp.GetId())
		assert.NotNil(t, resp.GetCreatedAt())
	})

	t.Run("пустые данные", func(t *testing.T) {
		_, err := testSecretClient.CreateSecret(ctx, &pb.CreateSecretRequest{Nonce: make([]byte, crypto.NonceSize)})
		assertCode(t, err, codes.InvalidArgument)
	})

	t.Run("без токена", func(t *testing.T) {
		_, err := testSecretClient.CreateSecret(context.Background(), validSecretReq())
		assertCode(t, err, codes.Unauthenticated)
	})
}

func TestHandler_GetSecret(t *testing.T) {
	ctx := authCtx(t, "grpc_secret_bob")

	created, err := testSecretClient.CreateSecret(ctx, validSecretReq())
	require.NoError(t, err)

	t.Run("найден", func(t *testing.T) {
		id := created.GetId()
		resp, err := testSecretClient.GetSecret(ctx, &pb.GetSecretRequest{Id: &id})
		require.NoError(t, err)
		assert.Equal(t, created.GetId(), resp.GetSecret().GetId())
	})

	t.Run("не найден", func(t *testing.T) {
		id := "00000000-0000-0000-0000-000000000000"
		_, err := testSecretClient.GetSecret(ctx, &pb.GetSecretRequest{Id: &id})
		assertCode(t, err, codes.NotFound)
	})

	t.Run("чужой секрет", func(t *testing.T) {
		otherCtx := authCtx(t, "grpc_secret_bob2")
		id := created.GetId()
		_, err := testSecretClient.GetSecret(otherCtx, &pb.GetSecretRequest{Id: &id})
		assertCode(t, err, codes.NotFound)
	})
}

func TestHandler_ListSecrets(t *testing.T) {
	ctx := authCtx(t, "grpc_secret_charlie")

	for range 3 {
		_, err := testSecretClient.CreateSecret(ctx, validSecretReq())
		require.NoError(t, err)
	}

	t.Run("все секреты", func(t *testing.T) {
		resp, err := testSecretClient.ListSecrets(ctx, &pb.ListSecretsRequest{})
		require.NoError(t, err)
		assert.Len(t, resp.GetSecrets(), 3)
	})
}

func TestHandler_UpdateSecret(t *testing.T) {
	ctx := authCtx(t, "grpc_secret_dave")

	created, err := testSecretClient.CreateSecret(ctx, validSecretReq())
	require.NoError(t, err)

	t.Run("успешное обновление", func(t *testing.T) {
		id := created.GetId()
		_, err := testSecretClient.UpdateSecret(ctx, &pb.UpdateSecretRequest{
			Id:    &id,
			Data:  []byte("updated-data"),
			Nonce: make([]byte, crypto.NonceSize),
		})
		require.NoError(t, err)
	})

	t.Run("не найден", func(t *testing.T) {
		id := "00000000-0000-0000-0000-000000000000"
		_, err := testSecretClient.UpdateSecret(ctx, &pb.UpdateSecretRequest{
			Id:    &id,
			Data:  []byte("data"),
			Nonce: make([]byte, crypto.NonceSize),
		})
		assertCode(t, err, codes.NotFound)
	})
}

func TestHandler_DeleteSecret(t *testing.T) {
	ctx := authCtx(t, "grpc_secret_eve")

	created, err := testSecretClient.CreateSecret(ctx, validSecretReq())
	require.NoError(t, err)

	t.Run("успешное удаление", func(t *testing.T) {
		id := created.GetId()
		_, err := testSecretClient.DeleteSecret(ctx, &pb.DeleteSecretRequest{Id: &id})
		require.NoError(t, err)
	})

	t.Run("не найден", func(t *testing.T) {
		id := "00000000-0000-0000-0000-000000000000"
		_, err := testSecretClient.DeleteSecret(ctx, &pb.DeleteSecretRequest{Id: &id})
		assertCode(t, err, codes.NotFound)
	})
}
