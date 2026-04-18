package grpcclient

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Gustik/trantor/api/gen/trantor/v1"
	"github.com/Gustik/trantor/internal/common/domain"
)

func TestToAuthError(t *testing.T) {
	tests := []struct {
		name    string
		code    codes.Code
		wantErr error
	}{
		{"NotFound", codes.NotFound, domain.ErrUserNotFound},
		{"AlreadyExists", codes.AlreadyExists, domain.ErrUserAlreadyExists},
		{"Unauthenticated", codes.Unauthenticated, domain.ErrInvalidCredentials},
		{"Internal", codes.Internal, domain.ErrInternal},
		{"Unknown", codes.Unknown, domain.ErrInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := toAuthError(status.Error(tt.code, ""))
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestToSecretError(t *testing.T) {
	tests := []struct {
		name    string
		code    codes.Code
		wantErr error
	}{
		{"NotFound", codes.NotFound, domain.ErrSecretNotFound},
		{"Unauthenticated", codes.Unauthenticated, domain.ErrInvalidCredentials},
		{"Internal", codes.Internal, domain.ErrInternal},
		{"Unknown", codes.Unknown, domain.ErrInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := toSecretError(status.Error(tt.code, ""))
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestProtoToSecret(t *testing.T) {
	t.Run("успешная конвертация", func(t *testing.T) {
		id := uuid.New()
		idStr := id.String()
		now := time.Now().UTC().Truncate(time.Second)

		s := &pb.Secret{
			Id:        &idStr,
			Data:      []byte("encrypted"),
			Nonce:     []byte("nonce"),
			CreatedAt: timestamppb.New(now),
			UpdatedAt: timestamppb.New(now),
		}

		secret, err := protoToSecret(s)
		require.NoError(t, err)
		assert.Equal(t, id, secret.ID)
		assert.Equal(t, []byte("encrypted"), secret.Data)
		assert.Equal(t, []byte("nonce"), secret.Nonce)
		assert.Equal(t, now, secret.CreatedAt)
		assert.Equal(t, now, secret.UpdatedAt)
	})

	t.Run("невалидный UUID", func(t *testing.T) {
		idStr := "not-a-uuid"
		s := &pb.Secret{Id: &idStr}

		_, err := protoToSecret(s)
		assert.ErrorIs(t, err, domain.ErrInternal)
	})
}
