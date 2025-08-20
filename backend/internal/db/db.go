package db

import (
	"context"
	"database/sql"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// DB wraps a sql.DB instance and exposes helper methods for common database
// operations. All queries should be executed via prepared statements to
// mitigate SQL injection vulnerabilities. The connection string should be
// provided via the DATABASE_URL environment variable. The recommended format
// is a PostgreSQL URL, for example:
//
//	postgres://username:password@host:port/database
//
// When using Supabase, copy the connection string from your project's settings.
type DB struct {
	*sql.DB
}

// New creates a new DB connection pool. It reads the DATABASE_URL
// environment variable and opens a pooled connection using pgx's stdlib
// driver. If the variable is not set or the connection fails, the
// application will log and exit. The returned *DB should be closed
// gracefully on shutdown.
func New() *DB {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable not set")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	// Set connection pool parameters. Adjust these based on your hosting
	// environment's limits (e.g. Supabase free tier supports up to 10 connections).
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	return &DB{db}
}

// Ping verifies a connection to the database can be established. It's a
// convenience method for health checks or startup verification.
func (d *DB) Ping(ctx context.Context) error {
	return d.DB.PingContext(ctx)
}

// Close gracefully closes the underlying sql.DB. Always call this on
// application shutdown to release connections back to the pool.
func (d *DB) Close() error {
	return d.DB.Close()
}
