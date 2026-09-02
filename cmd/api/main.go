package main

import (
	"context"
	"fmt"
	"github.com/nabbil/visionops/internal/visionops"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func rejectDemoSecrets(environment, jwtSecret, ingestKey string) error {
	if environment != "production" {
		return nil
	}
	if jwtSecret == "local-development-secret-change-me" || len(jwtSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be a unique value of at least 32 characters when APP_ENV=production")
	}
	if ingestKey == "vo_demo_ingest" || len(ingestKey) < 20 {
		return fmt.Errorf("INGEST_API_KEY must be a unique value of at least 20 characters when APP_ENV=production")
	}
	return nil
}

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	jwtSecret := env("JWT_SECRET", "local-development-secret-change-me")
	ingestKey := env("INGEST_API_KEY", "vo_demo_ingest")
	if err := rejectDemoSecrets(env("APP_ENV", "development"), jwtSecret, ingestKey); err != nil {
		log.Error("unsafe production configuration", "error", err)
		os.Exit(1)
	}
	db, err := visionops.OpenDB(env("DATABASE_URL", "postgres://visionops:visionops@localhost:5433/visionops?sslmode=disable"))
	if err != nil {
		log.Error("database unavailable", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	app := &visionops.App{DB: db, Secret: jwtSecret, IngestKey: ingestKey, Hub: visionops.NewHub(), Log: log, AllowPrivateWebhookTargets: env("APP_ENV", "development") != "production"}
	if err := app.Migrate(context.Background()); err != nil {
		log.Error("migration failed", "error", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go app.StartWorker(ctx)
	srv := &http.Server{Addr: ":8080", Handler: app.Routes(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	go func() { <-ctx.Done(); srv.Shutdown(context.Background()) }()
	log.Info("visionops listening", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Error("server stopped", "error", err)
	}
}
