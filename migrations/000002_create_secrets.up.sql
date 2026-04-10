CREATE TABLE secrets (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    data       BYTEA NOT NULL,    -- AES-GCM(master_key, SecretPayload{Type, Name, Data, Metadata})
    nonce      BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
