package config

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"sort"
	"time"

	_ "github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// WaitForDatabase retries until MySQL accepts connections. The database
// container usually starts slower than the API.
func WaitForDatabase(ctx context.Context, cfg *Config, log *slog.Logger) error {
	db, err := sql.Open("mysql", cfg.DB.DSN(false))
	if err != nil {
		return err
	}
	defer db.Close()

	deadline := time.Now().Add(90 * time.Second)
	for {
		err = db.PingContext(ctx)
		if err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("database not reachable: %w", err)
		}
		log.Info("waiting for database", "error", err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func OpenDatabase(cfg *Config) (*gorm.DB, error) {
	db, err := gorm.Open(gormmysql.Open(cfg.DB.DSN(false)), &gorm.Config{
		Logger: gormlogger.New(log.New(os.Stdout, "", log.LstdFlags), gormlogger.Config{
			SlowThreshold:             500 * time.Millisecond,
			LogLevel:                  gormlogger.Warn,
			IgnoreRecordNotFoundError: true, // an expected outcome, not an error
			ParameterizedQueries:      true, // never log query values (emails, token hashes)
		}),
		TranslateError: true,
		NowFunc:        func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	return db, nil
}

// RunMigrations applies the *.sql files in files that haven't run yet, in
// filename order, and records each in schema_migrations.
func RunMigrations(ctx context.Context, cfg *Config, files fs.FS, log *slog.Logger) error {
	db, err := sql.Open("mysql", cfg.DB.DSN(true))
	if err != nil {
		return err
	}
	defer db.Close()

	// One connection, so the advisory lock covers every statement below and
	// two API instances starting together can't migrate at the same time.
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	var locked int
	if err := conn.QueryRowContext(ctx, `SELECT GET_LOCK('schema_migrations', 60)`).Scan(&locked); err != nil || locked != 1 {
		return fmt.Errorf("could not acquire migration lock: %v", err)
	}
	defer conn.ExecContext(context.WithoutCancel(ctx), `SELECT RELEASE_LOCK('schema_migrations')`)

	if _, err := conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) NOT NULL PRIMARY KEY,
		applied_at DATETIME(3) NOT NULL
	)`); err != nil {
		return err
	}

	names, err := fs.Glob(files, "*.sql")
	if err != nil {
		return err
	}
	sort.Strings(names)

	for _, name := range names {
		var applied int
		if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, name).Scan(&applied); err != nil {
			return err
		}
		if applied > 0 {
			continue
		}
		body, err := fs.ReadFile(files, name)
		if err != nil {
			return err
		}
		if _, err := conn.ExecContext(ctx, string(body)); err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err := conn.ExecContext(ctx, `INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`, name, time.Now().UTC()); err != nil {
			return err
		}
		log.Info("migration applied", "version", name)
	}
	return nil
}
