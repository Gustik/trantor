# Trantor

Менеджер паролей. Клиент-серверная система для безопасного хранения логинов, паролей, текстовых данных, бинарных данных и банковских карт.

## Архитектура

```
trantor/
├── cmd/
│   ├── server/
│   │   └── main.go                      — инициализация зависимостей, запуск gRPC сервера
│   └── client/
│       ├── main.go                      — инициализация зависимостей, регистрация команд
│       └── commands/
│           ├── root.go                  — корневая команда, глобальные флаги (--server, --config)
│           ├── auth.go                  — команды: register, login
│           └── secret.go                — команды: secret add, list, get, delete, sync
│
├── internal/
│   ├── config/
│   │   └── config.go                    — struct Config, Load() из env/файла
│   │
│   ├── domain/
│   │   ├── user.go                      — struct User, ErrUserNotFound, ErrUserAlreadyExists, ErrInvalidCredentials
│   │   └── secret.go                    — struct Secret, SecretType (login/text/binary/card), ErrSecretNotFound, ErrAccessDenied
│   │
│   ├── server/
│   │   ├── auth/
│   │   │   └── service.go               — Register, GetSalt, Login, bcrypt(auth_key) + interface userStorage
│   │   ├── secret/
│   │   │   └── service.go               — Create, GetByID, List, Update, Delete + interface secretStorage
│   │   └── grpc/
│   │       ├── handler.go               — gRPC handlers, перекладывает proto ↔ domain + interface authService, secretService
│   │       └── interceptor.go           — UnaryInterceptor: проверка JWT, userID в контекст
│   │
│   ├── storage/
│   │   └── postgres/
│   │       ├── storage.go               — struct Storage{db *pgxpool.Pool}, New()
│   │       ├── user.go                  — CreateUser, FindUserByLogin
│   │       └── secret.go                — CreateSecret, GetSecretByID, ListSecrets, UpdateSecret, DeleteSecret
│   │
│   └── client/
│       ├── service/
│       │   ├── service.go               — struct Service, New(), interface grpcClient, interface vault
│       │   ├── auth.go                  — Register, GetSalt, Login, сохранение токена
│       │   └── secret.go                — Create, Get, List, Delete, Sync (сервер ↔ vault)
│       ├── grpcclient/
│       │   └── client.go                — gRPC соединение, все вызовы к серверу
│       └── vault/
│           └── vault.go                 — локальный кэш SQLite: хранение секретов офлайн
│
├── pkg/
│   ├── crypto/
│   │   └── crypto.go                    — AES-256-GCM шифрование, Argon2 деривация ключа из пароля
│   └── jwt/
│       └── jwt.go                       — GenerateToken, ValidateToken
│
├── api/
│   └── trantor.proto                    — gRPC контракт: сервисы, сообщения
│
├── migrations/
│   ├── 001_create_users.sql             — таблица users
│   └── 002_create_secrets.sql           — таблица secrets
│
├── docker-compose.yml                   — PostgreSQL для локальной разработки
├── Makefile                             — build, test, proto, lint, migrate
└── go.mod                               — github.com/Gustik/trantor
```

## Безопасность

### Zero-knowledge архитектура

Сервер никогда не видит данные пользователя в открытом виде.

### Схема шифрования

Пароль никогда не покидает клиент. Сервер не может расшифровать данные пользователя даже при полном доступе к БД.

```
Регистрация:
1. пользователь вводит пароль
2. Argon2(пароль + salt) → 64 байта
   первые 32 байта → auth_key        // для аутентификации на сервере
   вторые 32 байта → encryption_key  // для шифрования мастер-ключа, не покидает клиент
3. генерируем случайный мастер-ключ (32 байта)
4. AES-256-GCM(encryption_key, мастер-ключ) → encrypted_master_key
5. на сервер отправляем: auth_key + encrypted_master_key + master_key_nonce + argon2_salt
6. сервер хранит bcrypt(auth_key) вместо пароля

Логин:
1. пользователь вводит пароль
2. Argon2(пароль + argon2_salt из сервера) → 64 байта
   первые 32 байта → auth_key
   вторые 32 байта → encryption_key
3. отправляем auth_key на сервер → сервер проверяет bcrypt(auth_key)
4. сервер возвращает encrypted_master_key
5. AES-256-GCM decrypt(encryption_key, encrypted_master_key) → мастер-ключ
6. мастер-ключ живёт в памяти клиента на время сессии

Смена пароля:
1. Argon2(новый пароль + новый salt) → новые auth_key + encryption_key
2. AES-256-GCM(новый encryption_key, тот же мастер-ключ) → новый encrypted_master_key
3. данные не перешифровываем — мастер-ключ не изменился

Что знает сервер:
- bcrypt(auth_key)      — не может получить пароль или encryption_key
- encrypted_master_key  — не может расшифровать (нет encryption_key)
- argon2_salt           — без пароля бесполезен

Что знает только клиент:
- пароль
- encryption_key
- мастер-ключ (в памяти на время сессии)
```

### Хранение данных

**Сервер (PostgreSQL)** — всё зашифровано мастер-ключом, сервер не знает содержимого:
- `data` — зашифрованный blob (логин+пароль, текст, бинарные данные, карта)
- `metadata` — зашифрована вместе с `data`

**Клиент (SQLite vault)** — локальный кэш на устройстве пользователя:
- `metadata` — в открытом виде, для быстрого локального поиска
- `data` — зашифрован мастер-ключом, расшифровывается только по запросу пользователя

### Таблица users

```sql
CREATE TABLE users (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login                 TEXT NOT NULL UNIQUE,
    auth_key_hash         TEXT NOT NULL,         -- bcrypt(auth_key), пароль сервер не знает
    encrypted_master_key  BYTEA NOT NULL,         -- AES-GCM(encryption_key, master_key)
    master_key_nonce      BYTEA NOT NULL,         -- nonce для расшифровки мастер-ключа
    argon2_salt           BYTEA NOT NULL,         -- salt для Argon2
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### Таблица secrets

Сервер знает только кому принадлежит секрет и когда изменён — больше ничего.

```sql
CREATE TABLE secrets (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    data       BYTEA NOT NULL,    -- AES-GCM(master_key, SecretPayload{Type, Name, Data, Metadata})
    nonce      BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

`SecretPayload` — структура которую шифрует клиент перед отправкой:

```go
type SecretPayload struct {
    Type     SecretType        // login_password, text, binary, bank_card
    Name     string            // человекочитаемое имя, например "mysite.com"
    Data     []byte            // сами данные
    Metadata map[string]string // произвольные метаданные
}
```
