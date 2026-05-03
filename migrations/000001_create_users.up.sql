CREATE TABLE users (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login                 TEXT NOT NULL UNIQUE,
    auth_key_hash         TEXT NOT NULL,          -- bcrypt(auth_key), пароль сервер не знает
    encrypted_master_key  BYTEA NOT NULL,         -- AES-GCM(encryption_key, master_key)
    master_key_nonce      BYTEA NOT NULL,         -- nonce для расшифровки мастер-ключа
    argon2_salt           BYTEA NOT NULL,         -- salt для Argon2
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);
