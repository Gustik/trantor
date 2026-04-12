package grpc

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/Gustik/trantor/api/gen/trantor/v1"
	"github.com/Gustik/trantor/internal/domain"
	"github.com/Gustik/trantor/pkg/crypto"
	"github.com/Gustik/trantor/pkg/jwt"
)

func (h *Handler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if req.GetLogin() == "" {
		return nil, status.Error(codes.InvalidArgument, "login is required")
	}
	if len(req.GetAuthKey()) != crypto.KeySize {
		return nil, status.Error(codes.InvalidArgument, "auth_key must be 32 bytes")
	}
	if len(req.GetEncryptedMasterKey()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "encrypted_master_key is required")
	}
	if len(req.GetMasterKeyNonce()) != crypto.NonceSize {
		return nil, status.Error(codes.InvalidArgument, "master_key_nonce must be 12 bytes")
	}
	if len(req.GetArgon2Salt()) != crypto.SaltSize {
		return nil, status.Error(codes.InvalidArgument, "argon2_salt must be 32 bytes")
	}

	user := &domain.User{
		ID:                 uuid.New(),
		Login:              req.GetLogin(),
		AuthKeyHash:        string(req.GetAuthKey()),
		EncryptedMasterKey: req.GetEncryptedMasterKey(),
		MasterKeyNonce:     req.GetMasterKeyNonce(),
		Argon2Salt:         req.GetArgon2Salt(),
	}

	if err := h.auth.Register(ctx, user); err != nil {
		if errors.Is(err, domain.ErrUserAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, "user already exists")
		}
		return nil, status.Error(codes.Internal, "internal server error")
	}

	token, err := jwt.GenerateToken(user.ID, h.jwtSecret)
	if err != nil {
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &pb.RegisterResponse{Token: &token}, nil
}

func (h *Handler) GetSalt(ctx context.Context, req *pb.GetSaltRequest) (*pb.GetSaltResponse, error) {
	if req.GetLogin() == "" {
		return nil, status.Error(codes.InvalidArgument, "login is required")
	}
	salt, err := h.auth.GetSalt(ctx, req.GetLogin())
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "internal server error")
	}
	return &pb.GetSaltResponse{Argon2Salt: salt}, nil
}

func (h *Handler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req.GetLogin() == "" {
		return nil, status.Error(codes.InvalidArgument, "login is required")
	}
	if len(req.GetAuthKey()) != crypto.KeySize {
		return nil, status.Error(codes.InvalidArgument, "auth_key must be 32 bytes")
	}
	user, err := h.auth.Login(ctx, req.GetLogin(), req.GetAuthKey())
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		if errors.Is(err, domain.ErrInvalidCredentials) {
			return nil, status.Error(codes.Unauthenticated, "wrong credentials")
		}
		return nil, status.Error(codes.Internal, "internal server error")
	}

	token, err := jwt.GenerateToken(user.ID, h.jwtSecret)
	if err != nil {
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &pb.LoginResponse{
		Token:              &token,
		EncryptedMasterKey: user.EncryptedMasterKey,
		MasterKeyNonce:     user.MasterKeyNonce,
	}, nil
}
