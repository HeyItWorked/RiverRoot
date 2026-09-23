package store

import (
	"database/sql"
	"errors"
	"log"

	"github.com/HeyItWorked/riverroot/internal/pipeline"
	"github.com/google/uuid"
	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

// SQLiteStore implements BuildStore using SQLite instead of JSON files.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens (or creates) a SQLite database at dbPath.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// SQLite does NOT enforce foreign keys unless you flip this per-connection.
	if _, err = db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, err
	}

	if err = createTables(db); err != nil {
		db.Close()
		return nil, err
	}

	return &SQLiteStore{db: db}, nil
}

// createTables sets up the schema.
func createTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS builds (
			id         TEXT PRIMARY KEY,
			pipeline   TEXT NOT NULL,
			failed     BOOLEAN NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS steps (
			id        INTEGER PRIMARY KEY,
			build_id  TEXT NOT NULL REFERENCES builds(id),
			seq       INTEGER NOT NULL,
			name      TEXT NOT NULL,
			exit_code INTEGER NOT NULL,
			stdout    TEXT NOT NULL DEFAULT '',
			stderr    TEXT NOT NULL DEFAULT '',
			error     TEXT
		);
	`)
	return err
}

// Save persists a build and all its steps in a single transaction.
func (s *SQLiteStore) Save(result pipeline.BuildResult) (string, error) {
	id := uuid.New().String()
	transaction, err := s.db.Begin()
	if err != nil {
		return "", err
	}
	defer func() {
		if err := transaction.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			log.Printf("sqlite rollback failed: %v", err)
		}
	}()

	if _, err = transaction.Exec(
		"INSERT INTO builds (id, pipeline, failed) VALUES (?, ?, ?)",
		id, result.Pipeline, result.Failed,
	); err != nil {
		return "", err
	}

	for i, step := range result.Steps {
		var errStr *string
		if step.Err != nil {
			msg := step.Err.Error()
			errStr = &msg
		}
		if _, err = transaction.Exec(
			"INSERT INTO steps (build_id, seq, name, exit_code, stdout, stderr, error) VALUES (?, ?, ?, ?, ?, ?, ?)",
			id, i, step.Name, step.ExitCode, step.Stdout, step.Stderr, errStr,
		); err != nil {
			return "", err
		}
	}

	return id, transaction.Commit()
}

// Get retrieves a single build and its steps by ID.
func (s *SQLiteStore) Get(id string) (pipeline.BuildResult, error) {
	var result pipeline.BuildResult
	row := s.db.QueryRow("SELECT pipeline, failed, created_at FROM builds WHERE id = ?", id)
	if err := row.Scan(&result.Pipeline, &result.Failed, &result.CreatedAt); err != nil {
		return pipeline.BuildResult{}, err
	}
	result.ID = id
	steps, err := s.querySteps(id)
	if err != nil {
		return pipeline.BuildResult{}, err
	}
	result.Steps = steps
	return result, nil
}

// List returns all builds with their steps, newest first.
func (s *SQLiteStore) List() ([]pipeline.BuildResult, error) {
	rows, err := s.db.Query("SELECT id, pipeline, failed, created_at FROM builds ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []pipeline.BuildResult
	for rows.Next() {
		var id string
		var result pipeline.BuildResult
		if err = rows.Scan(&id, &result.Pipeline, &result.Failed, &result.CreatedAt); err != nil {
			return nil, err
		}
		result.ID = id
		steps, err := s.querySteps(id)
		if err != nil {
			return nil, err
		}
		result.Steps = steps
		results = append(results, result)
	}
	return results, rows.Err()
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// querySteps fetches all steps for a given build, ordered by seq.
func (s *SQLiteStore) querySteps(buildID string) ([]pipeline.StepResult, error) {
	rows, err := s.db.Query(
		"SELECT name, exit_code, stdout, stderr, error FROM steps WHERE build_id = ? ORDER BY seq",
		buildID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []pipeline.StepResult
	for rows.Next() {
		var step pipeline.StepResult
		var errStr sql.NullString
		if err = rows.Scan(&step.Name, &step.ExitCode, &step.Stdout, &step.Stderr, &errStr); err != nil {
			return nil, err
		}
		if errStr.Valid {
			step.Err = errors.New(errStr.String)
		}
		steps = append(steps, step)
	}
	return steps, rows.Err()
}
