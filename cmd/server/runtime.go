package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"timber-stage-qualifier/internal/application"
	"timber-stage-qualifier/internal/evidence"
	"timber-stage-qualifier/internal/httpapi"
	"timber-stage-qualifier/internal/repository"
)

type runtime struct {
	repository *repository.SQLiteRepository
	server     *http.Server
}

func buildRuntime(ctx context.Context, address, databasePath string) (*runtime, error) {
	dsn := databasePath
	if databasePath == ":memory:" {
		dsn = "file:selfcheck?mode=memory&cache=shared"
	}
	repo, err := repository.Open(ctx, dsn)
	if err != nil {
		return nil, err
	}
	service := application.NewService(repo, evidence.NewGenerator())
	api := httpapi.New(service)
	return &runtime{repository: repo, server: httpapi.NewHTTPServer(address, api.Handler())}, nil
}

func (r *runtime) close() error { return r.repository.Close() }

func runServer(ctx context.Context, c config) error {
	instance, err := buildRuntime(ctx, c.Address, c.DatabasePath)
	if err != nil {
		return fmt.Errorf("装配服务: %w", err)
	}
	defer instance.close()
	listener, err := net.Listen("tcp", c.Address)
	if err != nil {
		return fmt.Errorf("监听 %s: %w", c.Address, err)
	}
	signals, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	serveErr := make(chan error, 1)
	go func() {
		err := instance.server.Serve(listener)
		if err == http.ErrServerClosed {
			err = nil
		}
		serveErr <- err
	}()
	select {
	case err := <-serveErr:
		return err
	case <-signals.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := instance.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("关闭 HTTP 服务: %w", err)
		}
		return <-serveErr
	}
}
