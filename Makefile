MIGRATE=migrate -path migrations -database "postgres://trantor:trantor@localhost:5432/trantor?sslmode=disable"

.PHONY: build-server build-client build test test-integration lint proto migrate

# Сборка сервера
build-server:
	go build -o bin/server ./cmd/server

# Сборка клиента
build-client:
	go build -o bin/client ./cmd/client

# Сборка обоих бинарников
build: build-server build-client

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
	protoc --go_out=. --go-grpc_out=. api/trantor.proto

# Генерация документации из proto
docs:
	protoc --doc_out=./docs --doc_opt=markdown,api.md api/trantor.proto

# Применение всех миграций
migrate:
	$(MIGRATE) up

# Откат последней миграции
migrate-down:
	$(MIGRATE) down

# Откат всех миграций
migrate-drop:
	$(MIGRATE) drop
