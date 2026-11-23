package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/example/robot-fleet-learning-platform/orchestrator/internal/grpcapi"
	"github.com/example/robot-fleet-learning-platform/orchestrator/internal/storage"
)

func main() {
	addr := env("FLEET_ADDR", ":8080")
	db := env("FLEET_DB", "./data/fleet.db")
	if e := os.MkdirAll(filepath.Dir(db), 0755); e != nil {
		slog.Error("create data directory", "error", e)
		os.Exit(1)
	}
	st, e := storage.Open(db)
	if e != nil {
		slog.Error("open database", "error", e)
		os.Exit(1)
	}
	defer st.Close()
	api := grpcapi.New(st)
	if e = api.Recover(context.Background()); e != nil {
		slog.Error("recover jobs", "error", e)
		os.Exit(1)
	}
	srv := &http.Server{Addr: addr, Handler: api.Mux, ReadHeaderTimeout: 5 * time.Second}
	c, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go api.Maintenance(c)
	go func() {
		slog.Info("orchestrator listening", "address", addr)
		if e := srv.ListenAndServe(); e != nil && e != http.ErrServerClosed {
			slog.Error("http server failed", "error", e)
			stop()
		}
	}()
	<-c.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if e = srv.Shutdown(ctx); e != nil {
		slog.Error("shutdown failed", "error", e)
	}
}
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
