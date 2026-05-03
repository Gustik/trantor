CREATE TABLE secrets (
    id         UUID        PRIMARY KEY,
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    data       BYTEA,                                                          -- NULL если секрет удалён
    nonce      BYTEA,                                                          -- NULL если секрет удалён
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ                                                     -- NULL если секрет активен
);
