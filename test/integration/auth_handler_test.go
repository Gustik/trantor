//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/Gustik/trantor/api/gen/trantor/v1"
	"github.com/Gustik/trantor/pkg/crypto"
)

var validAuthKey = make([]byte, crypto.KeySize)
var validSalt = make([]byte, crypto.SaltSize)
var validNonce = make([]byte, crypto.NonceSize)
var validEncryptedMasterKey = []byte("encrypted-master-key-blob")

func validRegisterReq(login string) *pb.RegisterRequest {
	return &pb.RegisterRequest{
		Login:              &login,
		AuthKey:            validAuthKey,
		EncryptedMasterKey: validEncryptedMasterKey,
		MasterKeyNonce:     validNonce,
		Argon2Salt:         validSalt,
	}
}

func TestHandler_Register(t *testing.T) {
	ctx := context.Background()

	t.Run("успешная регистрация", func(t *testing.T) {
		resp, err := testAuthClient.Register(ctx, validRegisterReq("grpc_alice"))
		require.NoError(t, err)
		assert.NotEmpty(t, resp.GetToken())
	})

	t.Run("дублирующий логин", func(t *testing.T) {
		require.NoError(t, err(testAuthClient.Register(ctx, validRegisterReq("grpc_bob"))))
		_, err := testAuthClient.Register(ctx, validRegisterReq("grpc_bob"))
		assertCode(t, err, codes.AlreadyExists)
	})

	t.Run("пустой логин", func(t *testing.T) {
		req := validRegisterReq("")
		_, err := testAuthClient.Register(ctx, req)
		assertCode(t, err, codes.InvalidArgument)
	})

	t.Run("неверный размер auth_key", func(t *testing.T) {
		login := "grpc_charlie"
		req := &pb.RegisterRequest{Login: &login, AuthKey: []byte("short")}
		_, err := testAuthClient.Register(ctx, req)
		assertCode(t, err, codes.InvalidArgument)
	})
}

func TestHandler_GetSalt(t *testing.T) {
	ctx := context.Background()

	t.Run("найден", func(t *testing.T) {
		require.NoError(t, err(testAuthClient.Register(ctx, validRegisterReq("grpc_salt_user"))))
		login := "grpc_salt_user"
		resp, err := testAuthClient.GetSalt(ctx, &pb.GetSaltRequest{Login: &login})
		require.NoError(t, err)
		assert.NotEmpty(t, resp.GetArgon2Salt())
	})

	t.Run("пустой логин", func(t *testing.T) {
		login := ""
		_, err := testAuthClient.GetSalt(ctx, &pb.GetSaltRequest{Login: &login})
		assertCode(t, err, codes.InvalidArgument)
	})

	t.Run("не найден", func(t *testing.T) {
		login := "grpc_nobody"
		_, err := testAuthClient.GetSalt(ctx, &pb.GetSaltRequest{Login: &login})
		assertCode(t, err, codes.NotFound)
	})
}

func TestHandler_Login(t *testing.T) {
	ctx := context.Background()

	require.NoError(t, err(testAuthClient.Register(ctx, validRegisterReq("grpc_login_user"))))

	t.Run("успешный логин", func(t *testing.T) {
		login := "grpc_login_user"
		resp, err := testAuthClient.Login(ctx, &pb.LoginRequest{Login: &login, AuthKey: validAuthKey})
		require.NoError(t, err)
		assert.NotEmpty(t, resp.GetToken())
		assert.NotEmpty(t, resp.GetEncryptedMasterKey())
		assert.NotEmpty(t, resp.GetMasterKeyNonce())
	})

	t.Run("неверный auth_key", func(t *testing.T) {
		login := "grpc_login_user"
		wrongKey := make([]byte, crypto.KeySize)
		wrongKey[0] = 0xFF
		_, err := testAuthClient.Login(ctx, &pb.LoginRequest{Login: &login, AuthKey: wrongKey})
		assertCode(t, err, codes.Unauthenticated)
	})

	t.Run("пользователь не найден", func(t *testing.T) {
		login := "grpc_ghost"
		_, err := testAuthClient.Login(ctx, &pb.LoginRequest{Login: &login, AuthKey: validAuthKey})
		assertCode(t, err, codes.NotFound)
	})
}

// err извлекает ошибку из пары (response, error) для использования в require.NoError.
func err(_ interface{}, e error) error { return e }

// assertCode проверяет что ошибка имеет ожидаемый gRPC-код.
func assertCode(t *testing.T, err error, code codes.Code) {
	t.Helper()
	require.Error(t, err)
	assert.Equal(t, code, status.Code(err))
}
