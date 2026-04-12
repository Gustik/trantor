//go:build integration

package integration

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/Gustik/trantor/internal/config"
	"github.com/Gustik/trantor/internal/server/auth"
	"github.com/Gustik/trantor/internal/server/secret"
	pgstore "github.com/Gustik/trantor/internal/storage/postgres"
)

var (
	testStore          *pgstore.Storage
	testAuthService    *auth.Service
	testSecretService  *secret.Service
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("trantor"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		panic("start postgres container: " + err.Error())
	}

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic("get connection string: " + err.Error())
	}

	pgx5DSN := strings.Replace(dsn, "postgres://", "pgx5://", 1)
	mig, err := migrate.New("file://../../migrations", pgx5DSN)
	if err != nil {
		panic("create migrator: " + err.Error())
	}
	if err := mig.Up(); err != nil && err != migrate.ErrNoChange {
		panic("run migrations: " + err.Error())
	}

	cfg := &config.DBConfig{
		DSN:             dsn,
		MaxConns:        5,
		MinConns:        1,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
	}
	testStore, err = pgstore.New(ctx, cfg)
	if err != nil {
		panic("create storage: " + err.Error())
	}

	testAuthService = auth.New(testStore)
	testSecretService = secret.New(testStore)

	code := m.Run()

	testStore.Close()
	_ = pgContainer.Terminate(ctx)
	os.Exit(code)
}
