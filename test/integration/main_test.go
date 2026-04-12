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

	"net"

	pb "github.com/Gustik/trantor/api/gen/trantor/v1"
	"github.com/Gustik/trantor/internal/config"
	"github.com/Gustik/trantor/internal/server/auth"
	grpchandler "github.com/Gustik/trantor/internal/server/grpc"
	"github.com/Gustik/trantor/internal/server/secret"
	pgstore "github.com/Gustik/trantor/internal/storage/postgres"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var testJWTSecret = []byte("test-jwt-secret-32-bytes-long!!!")

var (
	testStore         *pgstore.Storage
	testAuthService   *auth.Service
	testSecretService *secret.Service
	testAuthClient    pb.AuthServiceClient
	testSecretClient  pb.SecretServiceClient
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

	// запускаем gRPC-сервер на случайном порту
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		panic("listen: " + err.Error())
	}
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpchandler.AuthInterceptor(testJWTSecret)),
	)
	handler := grpchandler.New(testAuthService, testSecretService, testJWTSecret)
	pb.RegisterAuthServiceServer(grpcServer, handler)
	pb.RegisterSecretServiceServer(grpcServer, handler)
	go grpcServer.Serve(lis)

	// создаём клиентов
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic("dial grpc: " + err.Error())
	}
	testAuthClient = pb.NewAuthServiceClient(conn)
	testSecretClient = pb.NewSecretServiceClient(conn)

	code := m.Run()

	grpcServer.GracefulStop()
	conn.Close()
	testStore.Close()
	_ = pgContainer.Terminate(ctx)
	os.Exit(code)
}
