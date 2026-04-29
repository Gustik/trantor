# Trantor

Менеджер паролей с E2E-шифрованием. Клиент-серверная система для безопасного хранения логинов, паролей, текстовых заметок, бинарных файлов и банковских карт.

## Архитектура

```
trantor/
├── cmd/
│   ├── server/
│   │   └── main.go                      — инициализация зависимостей, запуск gRPC сервера
│   └── client/
│       ├── main.go                      — флаг --version, запуск TUI
│       └── tui/
│           ├── tui.go                   — Start(): инициализация deps, запуск bubbletea
│           ├── root.go                  — rootModel: state machine, переходы между экранами
│           ├── auth.go                  — экраны: ввод пароля, login/register
│           ├── list.go                  — экран списка секретов, sync, logout
│           ├── detail.go                — экран просмотра секрета, удаление
│           ├── create.go                — экран создания секрета
│           ├── msgs.go                  — typed Msg для переходов между экранами
│           └── styles.go                — lipgloss: цвета, бейджи типов
│
├── internal/
│   ├── common/                          — общий код клиента и сервера
│   │   ├── config/
│   │   │   └── config.go                — ServerConfig, ClientConfig, DBConfig (cleanenv)
│   │   └── domain/
│   │       ├── user.go                  — ErrUserNotFound, ErrUserAlreadyExists, ErrInvalidCredentials
│   │       └── errors.go                — ErrInternal
│   │
│   ├── server/
│   │   ├── domain/
│   │   │   ├── user.go                  — User (серверная сущность)
│   │   │   └── secret.go                — Secret (ID, UserID, Data, Nonce, timestamps), ErrSecretNotFound
│   │   ├── auth/
│   │   │   └── service.go               — Register (bcrypt hash), GetSalt, Login
│   │   ├── secret/
│   │   │   └── service.go               — Create, GetByID, List, Update, Delete
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
│       │   ├── secret.go                — SecretType, SecretPayload, Secret, ErrSecretNotFound
│       │   └── errors.go                — ErrInternal, ErrNotAuthenticated
│       ├── auth/
│       │   └── service.go               — Register, Login, DeriveFromCache (Argon2 + крипто)
│       ├── secret/
│       │   └── service.go               — Create, Get, List, Delete, Sync
│       ├── grpcclient/
│       │   └── client.go                — gRPC-соединение, все вызовы к серверу
│       └── storage/
│           └── vault.go                 — SQLite: секреты, токен, auth cache, время синхронизации
│
├── pkg/
│   ├── crypto/
│   │   └── crypto.go                    — AES-256-GCM, Argon2, GenerateSalt, GenerateMasterKey
│   └── jwt/
│       └── jwt.go                       — GenerateToken, ValidateToken
│
├── api/
│   └── trantor.proto                    — gRPC контракт: AuthService, SecretService
│
├── test/
│   └── integration/                     — интеграционные тесты (testcontainers-go + реальный gRPC)
│
├── migrations/
│   ├── 000001_create_users.up.sql
│   └── 000002_create_secrets.up.sql
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

Повторный вход (с того же устройства, без сети):
1. пароль → Argon2 с кэшированным salt → encryption_key
2. расшифровываем кэшированный encrypted_master_key локально

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

**Сервер (PostgreSQL)** — сервер не знает содержимого секретов:
- `data` — весь `SecretPayload` зашифрован мастер-ключом целиком
- `nonce` — для расшифровки, `NULL` у удалённых секретов

**Клиент (SQLite vault)** — локальный кэш на устройстве:
- `type`, `name`, `metadata` — plaintext, для быстрого поиска
- `data` — зашифрован мастер-ключом, расшифровывается по запросу

### Схема таблиц

```sql
CREATE TABLE users (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login                 TEXT NOT NULL UNIQUE,
    auth_key_hash         TEXT NOT NULL,          -- bcrypt(auth_key)
    encrypted_master_key  BYTEA NOT NULL,         -- AES-GCM(encryption_key, master_key)
    master_key_nonce      BYTEA NOT NULL,
    argon2_salt           BYTEA NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE secrets (
    id         UUID        PRIMARY KEY,           -- назначается клиентом
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    data       BYTEA,                             -- NULL если удалён
    nonce      BYTEA,                             -- NULL если удалён
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ                        -- мягкое удаление
);
```

Удаление секрета — мягкое: `data` и `nonce` обнуляются, проставляется `deleted_at`. Клиент узнаёт об удалении при следующей синхронизации через параметр `updated_after`.

### Типы секретов

```go
type SecretPayload struct {
    Type     SecretType        // login_password | text | binary | bank_card
    Name     string            // человекочитаемое имя, например "mysite.com"
    Data     []byte            // сами данные (до 5 МБ для бинарных файлов)
    Metadata map[string]string // произвольные метаданные
}
```

## Синхронизация

Клиент работает в режиме offline-first: секреты создаются локально и отправляются на сервер при первой возможности. Sync двусторонний:

1. **Push** — отправляет секреты с `synced=false` на сервер
2. **Pull** — забирает с сервера секреты изменённые после `last_synced_at`

Конфликт двойного удаления: первый клиент получает OK, второй — `ErrSecretNotFound`. Локальный vault второго клиента очищается при следующем Sync через `deleted_at`.

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

### Клиент (TUI)

```bash
go run cmd/client/main.go
```

Навигация в TUI:

| Экран | Клавиша | Действие |
|---|---|---|
| Список | `↑↓` / `j k` | навигация |
| Список | `enter` | открыть секрет |
| Список | `n` | создать секрет |
| Список | `s` | синхронизировать с сервером |
| Список | `L` | выйти из аккаунта |
| Список | `q` | выйти из программы |
| Детали | `d` | удалить секрет |
| Детали | `esc` | назад |
| Везде | `ctrl+c` | выйти из программы |

При первом входе (пустой vault) автоматически выполняется синхронизация с сервером.

```bash
# Версия
go run cmd/client/main.go --version
```

## Тесты

```bash
# Юнит-тесты (без Docker)
go test ./...

# Интеграционные тесты (нужен Docker)
go test -tags=integration ./test/integration/...
```

## Миграции

```bash
migrate -path migrations -database "postgres://trantor:trantor@localhost:5432/trantor?sslmode=disable" up
```
