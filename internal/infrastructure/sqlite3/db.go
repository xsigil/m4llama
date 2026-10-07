package sqlite3

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func NewDB(ctx context.Context, dbPath string) (*sqlx.DB, error) {
	db, err := sqlx.ConnectContext(ctx, "sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}

	// SQLite の単一ライター特性に対応
	db.SetMaxOpenConns(1)

	// 初期マイグレーション
	schema := `
	CREATE TABLE IF NOT EXISTS samples (
		id TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);`
	if _, err := db.ExecContext(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to apply base schema: %w", err)
	}

	return db, nil
}
