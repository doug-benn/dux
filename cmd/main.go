package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/doug-benn/dux/internal/database"
	"github.com/doug-benn/dux/internal/middleware"
	"github.com/doug-benn/dux/internal/router"
	"github.com/golang-migrate/migrate/v4"
	"github.com/patrickmn/go-cache"
)

func main() {
	if err := run(os.Stdout, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(w io.Writer, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cache := cache.New(5*time.Minute, 10*time.Minute)

	sqliteDatabase, err := database.New(logger, "./db.sqlite3")
	if err != nil {
		logger.Error("Failed to create database", "error", err)
		os.Exit(1)
	}

	defer func() {
		if cerr := sqliteDatabase.Close(); cerr != nil {
			logger.Error("failed to close the database", "error", cerr)
		}
	}()

	if err = database.Migrate(sqliteDatabase); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logger.Error("failed to migrate database", "error", err)
		return nil
	}

	mux := http.NewServeMux()
	router.AddRoutes(mux, logger, cache, sqliteDatabase)

	handler := middleware.Recovery(logger)(mux)
	handler = middleware.AccessLogger(logger)(mux)

	//TODO load from env
	port := 8080

	// HTTP Server
	server := &http.Server{
		Addr:         fmt.Sprintf("127.0.0.1:%d", port),
		Handler:      handler,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	errChan := make(chan error, 1)

	//Main HTTP Server
	go func() {
		logger.Info("server started", "port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		logger.InfoContext(ctx, "shutting down server")

		// Create a new context for shutdown with timeout
		ctx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		// Shutdown the HTTP server first
		if err := server.Shutdown(ctx); err != nil {
			return fmt.Errorf("HTTP server shutdown: %w", err)
		}

		// cancel the main context
		cancel()

		return nil
	}
}
