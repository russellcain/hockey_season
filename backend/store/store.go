package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *sql.DB
}

func Open(dbPath, migrationsDir string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	s := &Store{db: db}
	if err := s.runMigrations(migrationsDir); err != nil {
		return nil, fmt.Errorf("migrations: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

// DB exposes the underlying connection for packages that need raw access (e.g. mockdata).
func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) runMigrations(dir string) error {
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (filename TEXT PRIMARY KEY)`); err != nil {
		return err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, name := range files {
		var existing string
		if err := s.db.QueryRow(`SELECT filename FROM schema_migrations WHERE filename = ?`, name).Scan(&existing); err == nil {
			continue
		}

		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}

		if _, err := s.db.Exec(string(content)); err != nil {
			return fmt.Errorf("exec %s: %w", name, err)
		}

		if _, err := s.db.Exec(`INSERT INTO schema_migrations (filename) VALUES (?)`, name); err != nil {
			return err
		}
	}
	return nil
}
