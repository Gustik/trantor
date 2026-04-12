package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Gustik/trantor/internal/domain"
	"github.com/google/uuid"
)


// CreateSecret сохраняет новый секрет в БД.
func (s *Storage) CreateSecret(ctx context.Context, secret *domain.Secret) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO secrets (id, user_id, data, nonce, created_at, updated_at)
		VALUES (@id, @user_id, @data, @nonce, @created_at, @updated_at)`,
		pgx.NamedArgs{
			"id":         secret.ID,
			"user_id":    secret.UserID,
			"data":       secret.Data,
			"nonce":      secret.Nonce,
			"created_at": secret.CreatedAt,
			"updated_at": secret.UpdatedAt,
		})
	if err != nil {
		return fmt.Errorf("create secret: %w", err)
	}
	return nil
}

// GetSecretByID возвращает секрет по ID и ID владельца.
// Возвращает ErrSecretNotFound если секрет не найден или принадлежит другому пользователю.
func (s *Storage) GetSecretByID(ctx context.Context, id, userID uuid.UUID) (*domain.Secret, error) {
	row := s.db.QueryRow(ctx,
		`SELECT id, user_id, data, nonce, created_at, updated_at
		FROM secrets WHERE id = $1 AND user_id = $2`,
		id, userID,
	)

	var secret domain.Secret
	err := row.Scan(
		&secret.ID,
		&secret.UserID,
		&secret.Data,
		&secret.Nonce,
		&secret.CreatedAt,
		&secret.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get secret: %w", err)
	}

	return &secret, nil
}

// ListSecrets возвращает все секреты пользователя изменённые после updatedAfter.
// Если updatedAfter равен нулю — возвращаются все секреты пользователя.
func (s *Storage) ListSecrets(ctx context.Context, userID uuid.UUID, updatedAfter time.Time) ([]*domain.Secret, error) {
	query := `SELECT id, user_id, data, nonce, created_at, updated_at
		FROM secrets WHERE user_id = $1`
	args := []any{userID}

	if !updatedAfter.IsZero() {
		query += ` AND updated_at > $2`
		args = append(args, updatedAfter)
	}

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}
	defer rows.Close()

	var secrets []*domain.Secret
	for rows.Next() {
		var secret domain.Secret
		if err := rows.Scan(
			&secret.ID,
			&secret.UserID,
			&secret.Data,
			&secret.Nonce,
			&secret.CreatedAt,
			&secret.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan secret: %w", err)
		}
		secrets = append(secrets, &secret)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}

	return secrets, nil
}

// UpdateSecret обновляет существующий секрет в БД.
func (s *Storage) UpdateSecret(ctx context.Context, secret *domain.Secret) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE secrets SET data = @data, nonce = @nonce, updated_at = @updated_at
		WHERE id = @id AND user_id = @user_id`,
		pgx.NamedArgs{
			"id":         secret.ID,
			"user_id":    secret.UserID,
			"data":       secret.Data,
			"nonce":      secret.Nonce,
			"updated_at": secret.UpdatedAt,
		})
	if err != nil {
		return fmt.Errorf("update secret: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteSecret удаляет секрет по ID и ID владельца.
func (s *Storage) DeleteSecret(ctx context.Context, id, userID uuid.UUID) error {
	tag, err := s.db.Exec(ctx,
		`DELETE FROM secrets WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	if err != nil {
		return fmt.Errorf("delete secret: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
