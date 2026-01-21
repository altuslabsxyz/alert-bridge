package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/altuslabsxyz/alert-bridge/internal/domain/repository"
)

//go:embed migrations/*.sql
var migrations embed.FS

// DB wraps a sql.DB connection with SQLite-specific functionality.
type DB struct {
	*sql.DB
	path string
}

// NewDB creates a new SQLite database connection.
// Use ":memory:" for an in-memory database.
func NewDB(path string) (*DB, error) {
	// Ensure directory exists for file-based database
	if path != ":memory:" {
		dir := filepath.Dir(path)
		if dir != "" && dir != "." {
			// Directory creation is handled by the caller or config
		}
	}

	// Build connection string with pragmas
	dsn := path
	if path != ":memory:" {
		dsn = fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=synchronous(NORMAL)", path)
	} else {
		dsn = "file::memory:?cache=shared&_pragma=foreign_keys(ON)"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// SQLite works best with single connection for writes
	db.SetMaxOpenConns(1)

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &DB{DB: db, path: path}, nil
}

// Migrate runs all pending database migrations.
func (db *DB) Migrate(ctx context.Context) error {
	// Check current schema version
	var currentVersion int
	err := db.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&currentVersion)
	if err != nil {
		// Table doesn't exist yet, that's fine
		currentVersion = 0
	}

	// Read and execute migration SQL
	data, err := migrations.ReadFile("migrations/001_initial.sql")
	if err != nil {
		return fmt.Errorf("read migration: %w", err)
	}

	// Only run if not already applied
	if currentVersion < 1 {
		_, err = db.ExecContext(ctx, string(data))
		if err != nil {
			return fmt.Errorf("execute migration: %w", err)
		}
	}

	return nil
}

// Close closes the database connection with proper cleanup.
func (db *DB) Close() error {
	// Force WAL checkpoint before close (only for file-based databases)
	if db.path != ":memory:" {
		_, _ = db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	}
	return db.DB.Close()
}

// Ping verifies the database connection is alive.
func (db *DB) Ping(ctx context.Context) error {
	return db.DB.PingContext(ctx)
}

// Path returns the database file path.
func (db *DB) Path() string {
	return db.path
}

// sqliteTx wraps sql.Tx to implement repository.Transaction
type sqliteTx struct {
	*sql.Tx
}

func (tx *sqliteTx) Commit() error {
	return tx.Tx.Commit()
}

func (tx *sqliteTx) Rollback() error {
	return tx.Tx.Rollback()
}

// BeginTx starts a new transaction.
func (db *DB) BeginTx(ctx context.Context) (repository.Transaction, error) {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	return &sqliteTx{Tx: tx}, nil
}

// WithTransaction executes a function within a transaction.
// If the function returns an error, the transaction is rolled back.
// Otherwise, the transaction is committed.
func (db *DB) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := db.BeginTx(ctx)
	if err != nil {
		return err
	}

	// Add transaction to context
	ctx = repository.NewContextWithTx(ctx, tx)

	// Execute function
	if err := fn(ctx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback after error %w: %v", err, rbErr)
		}
		return err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// getExecutor returns the appropriate executor (transaction or DB) from context.
func (db *DB) getExecutor(ctx context.Context) interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
} {
	if tx := repository.TxFromContext(ctx); tx != nil {
		if sqlTx, ok := tx.(*sqliteTx); ok {
			return sqlTx.Tx
		}
	}
	return db.DB
}
