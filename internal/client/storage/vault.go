// Package storage содержит локальное хранилище секретов на основе SQLite.
package storage

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Gustik/trantor/internal/client/domain"
)

// Vault реализует локальное хранилище секретов на основе SQLite.
type Vault struct {
	db *sql.DB
}

// New открывает или создаёт локальное хранилище по указанному пути.
func New(path string) (*Vault, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("open vault %s: %w", path, err)
	}

	v := &Vault{db: db}
	if err := v.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return v, nil
}

// SaveSecret сохраняет секрет локально.
func (v *Vault) SaveSecret(ctx context.Context, r *domain.Secret) error {
	meta, err := json.Marshal(r.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	syncedInt := 0
	if r.Synced {
		syncedInt = 1
	}

	_, err = v.db.ExecContext(ctx, `
		INSERT INTO secrets (id, type, name, data, data_nonce, metadata, updated_at, synced)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET
			type       = excluded.type,
			name       = excluded.name,
			data       = excluded.data,
			data_nonce = excluded.data_nonce,
			metadata   = excluded.metadata,
			updated_at = excluded.updated_at,
			synced     = excluded.synced
	`, r.ID.String(), r.Type, r.Name, r.Data, r.DataNonce, meta, r.UpdatedAt.Unix(), syncedInt)
	if err != nil {
		return fmt.Errorf("save secret: %w", err)
	}
	return nil
}

// MarkSynced помечает секрет как синхронизированный с сервером.
func (v *Vault) MarkSynced(ctx context.Context, id uuid.UUID) error {
	_, err := v.db.ExecContext(ctx, `UPDATE secrets SET synced = 1 WHERE id = ?`, id.String())
	if err != nil {
		return fmt.Errorf("mark synced: %w", err)
	}
	return nil
}

// ListUnsynced возвращает ID секретов, ещё не отправленных на сервер.
func (v *Vault) ListUnsynced(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := v.db.QueryContext(ctx, `SELECT id FROM secrets WHERE synced = 0`)
	if err != nil {
		return nil, fmt.Errorf("list unsynced: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var ids []uuid.UUID
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("scan id: %w", err)
		}
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("parse id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list unsynced: %w", err)
	}
	return ids, nil
}

// GetSecret возвращает локально сохранённый секрет по ID.
func (v *Vault) GetSecret(ctx context.Context, id uuid.UUID) (*domain.Secret, error) {
	row := v.db.QueryRowContext(ctx,
		`SELECT id, type, name, data, data_nonce, metadata, updated_at, synced FROM secrets WHERE id = ?`,
		id.String())

	r, err := scanSecret(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get secret: %w", ErrNotFound)
		}
		return nil, err
	}
	return r, nil
}

// ListSecrets возвращает все локально сохранённые секреты.
func (v *Vault) ListSecrets(ctx context.Context) ([]*domain.Secret, error) {
	rows, err := v.db.QueryContext(ctx,
		`SELECT id, type, name, data, data_nonce, metadata, updated_at, synced FROM secrets`)
	if err != nil {
		return nil, fmt.Errorf("execute query in list secrets: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var secrets []*domain.Secret
	for rows.Next() {
		r, err := scanSecret(rows.Scan)
		if err != nil {
			return nil, err
		}
		secrets = append(secrets, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}
	return secrets, nil
}

// DeleteSecret удаляет локально сохранённый секрет по serverID.
func (v *Vault) DeleteSecret(ctx context.Context, serverID uuid.UUID) error {
	_, err := v.db.ExecContext(ctx, `DELETE FROM secrets WHERE id = ?`, serverID.String())
	if err != nil {
		return fmt.Errorf("delete secret: %w", err)
	}
	return nil
}

// LastSyncedAt возвращает время последней успешной синхронизации с сервером.
func (v *Vault) LastSyncedAt(ctx context.Context) (time.Time, error) {
	var unix int64
	err := v.db.QueryRowContext(ctx, `SELECT value FROM meta WHERE key = 'last_synced_at'`).Scan(&unix)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("get last synced at: %w", err)
	}
	return time.Unix(unix, 0).UTC(), nil
}

// SetLastSyncedAt сохраняет время последней успешной синхронизации с сервером.
func (v *Vault) SetLastSyncedAt(ctx context.Context, t time.Time) error {
	_, err := v.db.ExecContext(ctx, `
		INSERT INTO meta (key, value) VALUES ('last_synced_at', ?)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value
	`, t.Unix())
	if err != nil {
		return fmt.Errorf("set last synced at: %w", err)
	}
	return nil
}

// SetAuthToken сохраняет токен авторизации.
func (v *Vault) SetAuthToken(ctx context.Context, token string) error {
	_, err := v.db.ExecContext(ctx, `
		INSERT INTO meta (key, value) VALUES ('auth_token', ?)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value
	`, token)
	if err != nil {
		return fmt.Errorf("set auth token: %w", err)
	}
	return nil
}

// GetAuthToken возвращает токен авторизации.
func (v *Vault) GetAuthToken(ctx context.Context) (string, error) {
	var token string
	err := v.db.QueryRowContext(ctx, `SELECT value FROM meta WHERE key = 'auth_token'`).Scan(&token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("get auth token: %w", ErrNotFound)
		}
		return "", fmt.Errorf("get auth token: %w", err)
	}
	return token, nil
}

// SetAuthCache сохраняет данные необходимые для восстановления мастер-ключа без обращения к серверу.
func (v *Vault) SetAuthCache(ctx context.Context, salt, encryptedMasterKey, masterKeyNonce []byte) error {
	entries := map[string][]byte{
		"argon2_salt":          salt,
		"encrypted_master_key": encryptedMasterKey,
		"master_key_nonce":     masterKeyNonce,
	}
	for key, val := range entries {
		_, err := v.db.ExecContext(ctx, `
			INSERT INTO meta (key, value) VALUES (?, ?)
			ON CONFLICT (key) DO UPDATE SET value = excluded.value
		`, key, base64.StdEncoding.EncodeToString(val))
		if err != nil {
			return fmt.Errorf("set auth cache %s: %w", key, err)
		}
	}
	return nil
}

// GetAuthCache возвращает данные для восстановления мастер-ключа.
func (v *Vault) GetAuthCache(ctx context.Context) (salt, encryptedMasterKey, masterKeyNonce []byte, err error) {
	keys := []string{"argon2_salt", "encrypted_master_key", "master_key_nonce"}
	result := make(map[string][]byte, len(keys))
	for _, key := range keys {
		var encoded string
		if err := v.db.QueryRowContext(ctx, `SELECT value FROM meta WHERE key = ?`, key).Scan(&encoded); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil, nil, ErrNotFound
			}
			return nil, nil, nil, fmt.Errorf("get auth cache %s: %w", key, err)
		}
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("decode auth cache %s: %w", key, err)
		}
		result[key] = decoded
	}
	return result["argon2_salt"], result["encrypted_master_key"], result["master_key_nonce"], nil
}

// Clear удаляет все данные из vault — используется при logout.
func (v *Vault) Clear(ctx context.Context) error {
	_, err := v.db.ExecContext(ctx, `DELETE FROM secrets; DELETE FROM meta;`)
	if err != nil {
		return fmt.Errorf("clear vault: %w", err)
	}
	return nil
}

// Close закрывает бд
func (v *Vault) Close() error {
	return v.db.Close()
}

// scanSecret сканирует строку из БД в cdomain.Secret.
func scanSecret(scan func(...any) error) (*domain.Secret, error) {
	var metaJSON []byte
	var idRaw string
	var updatedAtUnix int64
	var syncedInt int
	r := &domain.Secret{}

	err := scan(&idRaw, &r.Type, &r.Name, &r.Data, &r.DataNonce, &metaJSON, &updatedAtUnix, &syncedInt)
	if err != nil {
		return nil, fmt.Errorf("scan secret: %w", err)
	}
	id, err := uuid.Parse(idRaw)
	if err != nil {
		return nil, fmt.Errorf("parse id: %w", err)
	}
	if err := json.Unmarshal(metaJSON, &r.Metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}
	r.ID = id
	r.UpdatedAt = time.Unix(updatedAtUnix, 0).UTC()
	r.Synced = syncedInt == 1
	return r, nil
}

// migrate создаёт таблицы если они не существуют.
func (v *Vault) migrate(ctx context.Context) error {
	_, err := v.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS secrets (
			id         TEXT    PRIMARY KEY,
			type       TEXT    NOT NULL,
			name       TEXT    NOT NULL,
			data       BLOB    NOT NULL,
			data_nonce BLOB    NOT NULL,
			metadata   TEXT    NOT NULL DEFAULT '{}',
			updated_at INTEGER NOT NULL,
			synced     INTEGER NOT NULL DEFAULT 0
		);

		CREATE TABLE IF NOT EXISTS meta (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("vault migrate: %w", err)
	}
	return nil
}
