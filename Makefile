MIGRATE=migrate -path migrations -database "postgres://trantor:trantor@localhost:5432/trantor?sslmode=disable"

VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
BUILD_DATE = $(shell date -u +%Y-%m-%d)
LDFLAGS    = -X main.version=$(VERSION) -X main.buildDate=$(BUILD_DATE)

.PHONY: build-server build-client build run-server test test-integration lint proto migrate psql

# Сборка сервера
build-server:
	go build -o bin/server ./cmd/server

# Сборка клиента
build-client:
	go build -ldflags "$(LDFLAGS)" -o bin/client ./cmd/client

# Сборка обоих бинарников
build: build-server build-client

# Запуск сервера (поднимает postgres если не запущен, применяет миграции, стартует сервер)
run-server:
	docker compose up -d postgres
	$(MIGRATE) up
	TRANTOR_DSN="postgres://trantor:trantor@localhost:5432/trantor?sslmode=disable" \
	TRANTOR_JWT_SECRET="dev-secret-32-bytes-xxxxxxxxxxx" \
	go run ./cmd/server

# Кросс-компиляция клиента
build-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/client-linux ./cmd/client

build-darwin:
	GOOS=darwin GOARCH=amd64 go build -o bin/client-darwin ./cmd/client

build-windows:
	GOOS=windows GOARCH=amd64 go build -o bin/client-windows.exe ./cmd/client

# Тесты
test:
	go test ./...

# Интеграционные тесты (требуют Docker)
test-integration:
	go test -tags integration -v -timeout 120s ./test/integration/...

# Тесты с покрытием
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Линтер
lint:
	golangci-lint run ./...

# Генерация кода из proto
proto:
	protoc --go_out=. --go_opt=module=github.com/Gustik/trantor --go-grpc_out=. --go-grpc_opt=module=github.com/Gustik/trantor api/trantor.proto

# Генерация документации из proto
docs:
	protoc --doc_out=./docs --doc_opt=markdown,api.md api/trantor.proto

# Подключение к psql
psql:
	docker compose exec postgres psql -U trantor -d trantor

# Применение всех миграций
migrate:
	$(MIGRATE) up

# Откат последней миграции
migrate-down:
	$(MIGRATE) down

# Откат всех миграций
migrate-drop:
	$(MIGRATE) drop
