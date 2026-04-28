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
│           ├── root.go                  — корневая команда, глобальные флаги
│           ├── auth.go                  — команды: register, login
│           ├── secret.go                — команды: secret create, secret list
│           └── version.go               — команда: version
│
├── internal/
│   ├── common/                          — общий код клиента и сервера
│   │   ├── config/
│   │   │   └── config.go                — ServerConfig, ClientConfig, DBConfig (cleanenv)
│   │   └── domain/
│   │       ├── user.go                  — ErrUserNotFound, ErrUserAlreadyExists, ErrInvalidCredentials
│   │       ├── secret.go                — SecretType, SecretPayload, ErrSecretNotFound, ErrAccessDenied
│   │       └── errors.go                — ErrInternal
│   │
│   ├── server/
│   │   ├── domain/
│   │   │   ├── user.go                  — User (серверная сущность)
│   │   │   └── secret.go                — Secret (ID, UserID, Data, Nonce, CreatedAt, UpdatedAt, DeletedAt)
│   │   ├── auth/
│   │   │   └── service.go               — Register (bcrypt hash), GetSalt, Login + interface userStorage
│   │   ├── secret/
│   │   │   └── service.go               — Create, GetByID, List, Update, Delete + interface secretStorage
│   │   ├── storage/
│   │   │   ├── storage.go               — Storage{db *pgxpool.Pool}, New()
│   │   │   ├── errors.go                — ErrNotFound, ErrDuplicate
│   │   │   ├── user.go                  — CreateUser, FindUserByLogin, FindUserByID
│   │   │   └── secret.go                — CreateSecret, GetSecretByID, ListSecrets, UpdateSecret, DeleteSecret (мягкое удаление)
│   │   └── grpc/
│   │       ├── handler.go               — Handler struct, интерфейсы authService/secretService
│   │       ├── auth_handler.go          — Register, GetSalt, Login
│   │       ├── secret_handler.go        — CreateSecret, GetSecret, ListSecrets, UpdateSecret, DeleteSecret
│   │       └── interceptor.go           — AuthInterceptor (JWT), UserIDFromContext
│   │
│   └── client/
│       ├── domain/
│       │   ├── user.go                  — User (клиентская сущность)
│       │   ├── secret.go                — Secret (ID, Type, Name, Data, DataNonce, Metadata, Synced)
│       │   └── errors.go                — ErrSecretNotFound, ErrInternal
│       ├── auth/
│       │   └── service.go               — Register, Login (Argon2 + крипто, сохранение токена)
│       ├── secret/
│       │   └── service.go               — Create, List (шифрование/расшифровка, синхронизация)
│       ├── grpcclient/
│       │   └── client.go                — gRPC-соединение, все вызовы к серверу
│       └── storage/
│           └── vault.go                 — локальный кэш SQLite: секреты, токен, время синхронизации
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
├── test/
│   └── integration/                     — интеграционные тесты (testcontainers-go)
│
├── migrations/
│   ├── 000001_create_users.up.sql       — создание таблицы users
│   ├── 000001_create_users.down.sql
│   ├── 000002_create_secrets.up.sql     — создание таблицы secrets
│   └── 000002_create_secrets.down.sql
│
├── bin/                                 — собранные бинарники (gitignore)
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
- `type`, `name`, `metadata` — в открытом виде, для быстрого локального поиска
- `data` — зашифрован мастер-ключом, расшифровывается только по запросу пользователя

### Таблица users

```sql
CREATE TABLE users (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login                 TEXT NOT NULL UNIQUE,
    auth_key_hash         TEXT NOT NULL,          -- bcrypt(auth_key), пароль сервер не знает
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
    id         UUID        PRIMARY KEY,
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    data       BYTEA,                             -- NULL если секрет удалён
    nonce      BYTEA,                             -- NULL если секрет удалён
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ                        -- NULL если секрет активен
);
```

Удаление секрета — мягкое: `data` и `nonce` обнуляются, проставляется `deleted_at`. Клиент узнаёт об удалении при следующей синхронизации через `updated_after`.

`SecretPayload` — структура которую шифрует клиент перед отправкой:

```go
type SecretPayload struct {
    Type     SecretType        // login_password, text, binary, bank_card
    Name     string            // человекочитаемое имя, например "mysite.com"
    Data     []byte            // сами данные
    Metadata map[string]string // произвольные метаданные
}
```

## Конфигурация

### Сервер

| Переменная | Описание | По умолчанию |
|---|---|---|
| `TRANTOR_GRPC` | Адрес gRPC-сервера | `:50051` |
| `TRANTOR_JWT_SECRET` | Секрет JWT | обязателен |
| `TRANTOR_DSN` | PostgreSQL DSN | обязателен |
| `TRANTOR_DB_MAX_CONNS` | Макс. соединений в пуле | `10` |
| `TRANTOR_DB_MIN_CONNS` | Мин. соединений в пуле | `2` |
| `TRANTOR_DB_MAX_CONN_LIFETIME` | Макс. время жизни соединения | `1h` |
| `TRANTOR_DB_MAX_CONN_IDLE_TIME` | Макс. время простоя соединения | `30m` |

### Клиент

| Переменная | Описание | По умолчанию |
|---|---|---|
| `TRANTOR_SERVER_ADDR` | Адрес сервера | `localhost:50051` |
| `TRANTOR_VAULT_PATH` | Путь к локальному хранилищу | `~/.trantor/vault.db` |
| `TRANTOR_TLS` | Включить TLS | `false` |

## Запуск

### Сервер

```bash
TRANTOR_DSN="postgres://trantor:trantor@localhost:5432/trantor?sslmode=disable" \
TRANTOR_JWT_SECRET="some-random-secret-32-bytes-long" \
go run cmd/server/main.go
```

### Клиент

```bash
# Регистрация
go run cmd/client/main.go register --login user --password secret

# Вход
go run cmd/client/main.go login --login user --password secret

# Создать секрет
go run cmd/client/main.go secret create --name "mysite.com" --data "hunter2" --type login_password

# Список секретов (id, тип, имя)
go run cmd/client/main.go secret list

# Получить секрет по ID (выводит расшифрованные данные)
go run cmd/client/main.go secret get --id <uuid>

# Удалить секрет
go run cmd/client/main.go secret delete --id <uuid>

# Фоновая синхронизация с сервером каждые 10 секунд (Ctrl+C для остановки)
go run cmd/client/main.go secret sync

# Версия
go run cmd/client/main.go version
```

## Тесты

```bash
# Юнит-тесты (без Docker)
go test ./...

# Интеграционные тесты (нужен Docker)
go test -tags=integration ./test/integration/...
```

Пирамида тестов:
- **Юнит**: `pkg/crypto`, `pkg/jwt`, `internal/common/config`, `internal/server/auth`, `internal/server/secret` — мок-хранилища через testify/mock.
- **Интеграционные**: `test/integration/` — testcontainers-go (PostgreSQL), реальный gRPC-сервер на случайном порту.

## Миграции

Файлы: `migrations/`. Запускаются golang-migrate.

```bash
migrate -path migrations -database "postgres://trantor:trantor@localhost:5432/trantor?sslmode=disable" up
```

