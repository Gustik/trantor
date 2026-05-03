package grpc

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Gustik/trantor/api/gen/trantor/v1"
	"github.com/Gustik/trantor/internal/server/domain"
	"github.com/Gustik/trantor/pkg/crypto"
)

func (h *Handler) CreateSecret(ctx context.Context, req *pb.CreateSecretRequest) (*pb.CreateSecretResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if len(req.GetData()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "data is required")
	}
	if len(req.GetNonce()) != crypto.NonceSize {
		return nil, status.Error(codes.InvalidArgument, "nonce must be 12 bytes")
	}

	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	userID, ok := UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthenticated")
	}

	secret := &domain.Secret{
		ID:     id,
		UserID: userID,
		Data:   req.GetData(),
		Nonce:  req.GetNonce(),
	}

	if err := h.secret.Create(ctx, secret); err != nil {
		slog.ErrorContext(ctx, "create secret", "err", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	idStr := secret.ID.String()
	return &pb.CreateSecretResponse{
		Id:        &idStr,
		CreatedAt: timestamppb.New(secret.CreatedAt),
	}, nil
}

func (h *Handler) GetSecret(ctx context.Context, req *pb.GetSecretRequest) (*pb.GetSecretResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	userID, ok := UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthenticated")
	}

	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	secret, err := h.secret.GetByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, domain.ErrSecretNotFound) {
			return nil, status.Error(codes.NotFound, "secret not found")
		}
		slog.ErrorContext(ctx, "get secret", "err", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &pb.GetSecretResponse{Secret: secretToProto(secret)}, nil
}

func (h *Handler) ListSecrets(ctx context.Context, req *pb.ListSecretsRequest) (*pb.ListSecretsResponse, error) {
	userID, ok := UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthenticated")
	}

	var updatedAfter time.Time
	if req.GetUpdatedAfter() != nil {
		updatedAfter = req.GetUpdatedAfter().AsTime()
	}

	secrets, err := h.secret.List(ctx, userID, updatedAfter)
	if err != nil {
		slog.ErrorContext(ctx, "list secrets", "err", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	result := make([]*pb.Secret, len(secrets))
	for i, s := range secrets {
		result[i] = secretToProto(s)
	}

	return &pb.ListSecretsResponse{Secrets: result}, nil
}

func (h *Handler) UpdateSecret(ctx context.Context, req *pb.UpdateSecretRequest) (*pb.UpdateSecretResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if len(req.GetData()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "data is required")
	}
	if len(req.GetNonce()) != crypto.NonceSize {
		return nil, status.Error(codes.InvalidArgument, "nonce must be 12 bytes")
	}

	userID, ok := UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthenticated")
	}

	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	secret := &domain.Secret{
		ID:     id,
		UserID: userID,
		Data:   req.GetData(),
		Nonce:  req.GetNonce(),
	}

	if err := h.secret.Update(ctx, secret); err != nil {
		if errors.Is(err, domain.ErrSecretNotFound) {
			return nil, status.Error(codes.NotFound, "secret not found")
		}
		slog.ErrorContext(ctx, "update secret", "err", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &pb.UpdateSecretResponse{UpdatedAt: timestamppb.New(secret.UpdatedAt)}, nil
}

func (h *Handler) DeleteSecret(ctx context.Context, req *pb.DeleteSecretRequest) (*pb.DeleteSecretResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	userID, ok := UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthenticated")
	}

	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	if err := h.secret.Delete(ctx, id, userID); err != nil {
		if errors.Is(err, domain.ErrSecretNotFound) {
			return nil, status.Error(codes.NotFound, "secret not found")
		}
		slog.ErrorContext(ctx, "delete secret", "err", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &pb.DeleteSecretResponse{}, nil
}

// secretToProto конвертирует domain.Secret в proto-сообщение.
func secretToProto(s *domain.Secret) *pb.Secret {
	id := s.ID.String()
	result := &pb.Secret{
		Id:        &id,
		Data:      s.Data,
		Nonce:     s.Nonce,
		CreatedAt: timestamppb.New(s.CreatedAt),
		UpdatedAt: timestamppb.New(s.UpdatedAt),
	}
	if s.DeletedAt != nil {
		result.DeletedAt = timestamppb.New(*s.DeletedAt)
	}
	return result
}
