// Package main является точкой входа сервера Trantor.
package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	pb "github.com/Gustik/trantor/api/gen/trantor/v1"
	"github.com/Gustik/trantor/internal/common/config"
	"github.com/Gustik/trantor/internal/server/auth"
	grpchandler "github.com/Gustik/trantor/internal/server/grpc"
	"github.com/Gustik/trantor/internal/server/secret"
	"github.com/Gustik/trantor/internal/server/storage"
)

func main() {
	srvCfg, dbCfg := loadCfg()
	ctx, cancel := context.WithCancel(context.Background())

	db, err := storage.New(ctx, dbCfg)
	if err != nil {
		slog.Error("init DB error", "err", err)
		os.Exit(1)
	}

	authSvc := auth.New(db)
	secretSvc := secret.New(db)

	lis, err := net.Listen("tcp", srvCfg.GRPC)
	if err != nil {
		slog.Error("listen error", "addr", srvCfg.GRPC, "err", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpchandler.AuthInterceptor([]byte(srvCfg.JWTSecret))),
	)
	handler := grpchandler.New(authSvc, secretSvc, []byte(srvCfg.JWTSecret))

	pb.RegisterAuthServiceServer(grpcServer, handler)
	pb.RegisterSecretServiceServer(grpcServer, handler)
	go grpcServer.Serve(lis)
	slog.Info("Сервер запущен", "grpc", srvCfg.GRPC)

	gracefulStop(grpcServer, cancel, db)
}

func gracefulStop(g *grpc.Server, cancel context.CancelFunc, d *storage.Storage) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	sig := <-quit
	slog.Info("Получен сигнал выхода", "sig", sig)

	g.GracefulStop()
	cancel()
	d.Close()
}

func loadCfg() (*config.ServerConfig, *config.DBConfig) {
	srvCfg, err := config.LoadServer()
	if err != nil {
		slog.Error("load server config error", "err", err)
		os.Exit(1)
	}
	dbCfg, err := config.LoadDB()
	if err != nil {
		slog.Error("load DB config error", "err", err)
		os.Exit(1)
	}

	return srvCfg, dbCfg
}
