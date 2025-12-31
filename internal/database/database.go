package database

import (
	"log/slog"

	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/doug-benn/dux/internal/database/queries"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Database interface {
	DB() *sql.DB
	Queries() *queries.Queries
	Logger() *slog.Logger
	Close() error
}

func New(logger *slog.Logger, url string) (Database, error) {
	db, err := newLocalDB(logger, url)
	if err != nil {
		return nil, err
	}
	if err = db.DB().PingContext(context.Background()); err != nil {
		return nil, err
	}
	return db, nil
}

// Migrate runs the migrations on the database. Assumes the database is SQLite.
func Migrate(db Database) (err error) {
	driver, err := sqlite3.WithInstance(db.DB(), &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("failed to create database driver: %w", err)
	}

	iofsDriver, err := iofs.New(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create iofs: %w", err)
	}
	defer func() {
		if cerr := iofsDriver.Close(); cerr != nil {
			err = errors.Join(err, fmt.Errorf("failed to close driver: %w", cerr))
		}
	}()

	m, err := migrate.NewWithInstance("iofs", iofsDriver, "sqlite3", driver)
	if err != nil {
		return fmt.Errorf("failed to create migration: %w", err)
	}

	return m.Up()
}

type LocalDB struct {
	logger  *slog.Logger
	db      *sql.DB
	queries *queries.Queries
}

var _ Database = (*LocalDB)(nil)

func (d *LocalDB) DB() *sql.DB {
	return d.db
}

func (d *LocalDB) Queries() *queries.Queries {
	return d.queries
}

func (d *LocalDB) Logger() *slog.Logger {
	return d.logger
}

func (d *LocalDB) Close() error {
	return d.db.Close()
}

func newLocalDB(logger *slog.Logger, path string) (*LocalDB, error) {
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		return nil, err
	}
	return &LocalDB{logger: logger, db: db, queries: queries.New(db)}, nil
}
