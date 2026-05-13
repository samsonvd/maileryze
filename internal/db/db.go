package db

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"time"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"

	"maileryze/internal/cfg"
	"maileryze/internal/connector"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Open opens the SQLite database and runs any pending migrations.
func Open() (*sql.DB, error) {
	if err := os.MkdirAll(cfg.DataDir(), 0700); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}

	db, err := sql.Open("sqlite", cfg.DbFilePath())
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;`); err != nil {
		return nil, fmt.Errorf("setting pragmas: %w", err)
	}

	goose.SetBaseFS(migrations)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("sqlite3"); err != nil {
		return nil, fmt.Errorf("setting dialect: %w", err)
	}
	if err := goose.Up(db, "migrations"); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return db, nil
}

// InsertEmail inserts an email into the database. Duplicates (matched by
// alias + provider + provider_identifier) are silently ignored.
// Returns true if the row was newly inserted, false if it already existed.
func InsertEmail(db *sql.DB, alias, provider string, email connector.EmailContent[any]) (bool, error) {
	result, err := db.Exec(`
		INSERT OR IGNORE INTO emails
			(alias, subject, sender_name, sender_address, unsubscribe_email, unsubscribe_url, provider, provider_identifier, received_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		alias,
		email.Subject,
		email.Sender.Name,
		email.Sender.Address,
		email.Unsubscribe.Email,
		email.Unsubscribe.URL,
		provider,
		email.Provider.Identifier,
		email.ReceivedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows > 0, err
}
