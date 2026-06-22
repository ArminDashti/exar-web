package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

func Open(path string) (*DB, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create db directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &DB{DB: db}, nil
}

func migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS persons (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS shops (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE
	);

	CREATE TABLE IF NOT EXISTS invoices (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		person_id INTEGER NOT NULL,
		shop_id INTEGER NOT NULL,
		date TEXT NOT NULL,
		total REAL NOT NULL DEFAULT 0,
		FOREIGN KEY (person_id) REFERENCES persons(id),
		FOREIGN KEY (shop_id) REFERENCES shops(id)
	);

	CREATE TABLE IF NOT EXISTS invoice_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		invoice_id INTEGER NOT NULL,
		description TEXT NOT NULL,
		amount REAL NOT NULL,
		quantity REAL NOT NULL DEFAULT 1,
		FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_invoices_person ON invoices(person_id);
	CREATE INDEX IF NOT EXISTS idx_invoices_date ON invoices(date);
	CREATE INDEX IF NOT EXISTS idx_invoices_shop ON invoices(shop_id);
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("migrate schema: %w", err)
	}

	return seedPersons(db)
}

func seedPersons(db *sql.DB) error {
	persons := []struct {
		id   int
		name string
	}{
		{1, "Person 1"},
		{2, "Person 2"},
	}

	for _, p := range persons {
		_, err := db.Exec(
			`INSERT INTO persons (id, name) VALUES (?, ?)
			 ON CONFLICT(id) DO UPDATE SET name = excluded.name`,
			p.id, p.name,
		)
		if err != nil {
			return fmt.Errorf("seed person %d: %w", p.id, err)
		}
	}

	return nil
}
