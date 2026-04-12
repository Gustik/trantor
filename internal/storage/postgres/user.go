package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Gustik/trantor/internal/domain"
)

const pgDuplicateErrorCode = "23505"

// CreateUser сохраняет нового пользователя в БД.
// Возвращает ErrUserAlreadyExists если логин уже занят.
func (s *Storage) CreateUser(ctx context.Context, user *domain.User) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO users (id, login, auth_key_hash, encrypted_master_key, master_key_nonce, argon2_salt, created_at)
		VALUES (@id, @login, @auth_key_hash, @encrypted_master_key, @master_key_nonce, @argon2_salt, @created_at)`,
		pgx.NamedArgs{
			"id":                   user.ID,
			"login":                user.Login,
			"auth_key_hash":        user.AuthKeyHash,
			"encrypted_master_key": user.EncryptedMasterKey,
			"master_key_nonce":     user.MasterKeyNonce,
			"argon2_salt":          user.Argon2Salt,
			"created_at":           user.CreatedAt,
		})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgDuplicateErrorCode {
			return ErrDuplicate
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// FindUserByLogin возвращает пользователя по логину.
// Возвращает ErrUserNotFound если пользователь не найден.
func (s *Storage) FindUserByLogin(ctx context.Context, login string) (*domain.User, error) {
	row := s.db.QueryRow(ctx,
		`SELECT id, login, auth_key_hash, encrypted_master_key, master_key_nonce, argon2_salt, created_at 
		FROM users WHERE login = $1`,
		login,
	)

	var user domain.User
	err := row.Scan(
		&user.ID,
		&user.Login,
		&user.AuthKeyHash,
		&user.EncryptedMasterKey,
		&user.MasterKeyNonce,
		&user.Argon2Salt,
		&user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find user by login: %w", err)
	}

	return &user, nil
}

// FindUserByID возвращает пользователя по ID.
// Возвращает ErrUserNotFound если пользователь не найден.
func (s *Storage) FindUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	row := s.db.QueryRow(ctx,
		`SELECT id, login, auth_key_hash, encrypted_master_key, master_key_nonce, argon2_salt, created_at 
		FROM users WHERE id = $1`,
		id,
	)

	var user domain.User
	err := row.Scan(
		&user.ID,
		&user.Login,
		&user.AuthKeyHash,
		&user.EncryptedMasterKey,
		&user.MasterKeyNonce,
		&user.Argon2Salt,
		&user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	return &user, nil
}
