// Package vault содержит локальное хранилище секретов на основе SQLite.
package vault

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Gustik/trantor/internal/domain"
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
		db.Close()
		return nil, err
	}
	return v, nil
}

// SaveSecret сохраняет расшифрованный секрет локально.
// metadata хранится в открытом виде, data — зашифрован мастер-ключом.
func (v *Vault) SaveSecret(ctx context.Context, payload *domain.SecretPayload, serverID uuid.UUID, updatedAt time.Time) error {
	meta, err := json.Marshal(payload.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	_, err = v.db.ExecContext(ctx, `
          INSERT INTO secrets (server_id, type, name, data, metadata, updated_at)                                                                                                                                                                                                             
          VALUES (?, ?, ?, ?, ?, ?)           
          ON CONFLICT (server_id) DO UPDATE SET
              type       = excluded.type,
              name       = excluded.name,                                                                                                                                                                                                                                                     
              data       = excluded.data,     
              metadata   = excluded.metadata,                                                                                                                                                                                                                                                 
              updated_at = excluded.updated_at
      `, serverID.String(), payload.Type, payload.Name, payload.Data, meta, updatedAt.Unix())
	if err != nil {
		return fmt.Errorf("save secret: %w", err)
	}
	return nil
}

// GetSecret возвращает локально сохранённый секрет по serverID.
func (v *Vault) GetSecret(ctx context.Context, serverID uuid.UUID) (*domain.SecretPayload, error) {
	row := v.db.QueryRowContext(ctx, `SELECT type, name, data, metadata FROM secrets WHERE server_id = ?`,
		serverID.String())

	payload, err := scanSecret(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSecretNotFound
		}
		return nil, err
	}
	return payload, nil
}

// ListSecrets возвращает все локально сохранённые секреты.
func (v *Vault) ListSecrets(ctx context.Context) ([]*domain.SecretPayload, error) {
	rows, err := v.db.QueryContext(ctx, `SELECT type, name, data, metadata FROM secrets`)
	if err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}
	defer rows.Close()

	var result []*domain.SecretPayload
	for rows.Next() {
		payload, err := scanSecret(rows.Scan)
		if err != nil {
			return nil, err
		}
		result = append(result, payload)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}
	return result, nil
}

// DeleteSecret удаляет локально сохранённый секрет по serverID.
func (v *Vault) DeleteSecret(ctx context.Context, serverID uuid.UUID) error {
	_, err := v.db.ExecContext(ctx, `DELETE FROM secrets WHERE server_id = ?`, serverID.String())
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

// Close закрывает бд
func (v *Vault) Close() error {
	return v.db.Close()
}

// scanSecret сканирует строку из БД в SecretPayload.
func scanSecret(scan func(...any) error) (*domain.SecretPayload, error) {
	var metaJSON []byte
	payload := &domain.SecretPayload{}

	if err := scan(&payload.Type, &payload.Name, &payload.Data, &metaJSON); err != nil {
		return nil, fmt.Errorf("scan secret: %w", err)
	}
	if err := json.Unmarshal(metaJSON, &payload.Metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}
	return payload, nil
}

// migrate создаёт таблицы если они не существуют.
func (v *Vault) migrate(ctx context.Context) error {
	_, err := v.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS secrets (
			server_id  TEXT    PRIMARY KEY,
			type       TEXT    NOT NULL,
			name       TEXT    NOT NULL,
			data       BLOB    NOT NULL,
			metadata   TEXT    NOT NULL DEFAULT '{}',
			updated_at INTEGER NOT NULL
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
