package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// InitDB initializes SQLite, configures PRAGMAs (WAL mode, busy timeout, foreign keys),
// and executes the schema migration.
func InitDB(dbPath string) (*sql.DB, error) {
	isMemory := dbPath == ":memory:" || strings.Contains(dbPath, "mode=memory")
	if !isMemory {
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create db directory %s: %w", dir, err)
		}
	}

	dsn := dbPath
	if isMemory {
		if !strings.Contains(dsn, "?") {
			dsn = "file:memdb1?mode=memory&cache=shared&_foreign_keys=ON&_busy_timeout=5000"
		}
	} else {
		separator := "?"
		if strings.Contains(dsn, "?") {
			separator = "&"
		}
		dsn = fmt.Sprintf("%s%s_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000&_foreign_keys=ON", dsn, separator)
	}

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	if isMemory {
		db.SetMaxOpenConns(1)
	} else {
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(10)
	}

	// Explicitly apply PRAGMAs to guarantee foreign key enforcement and WAL
	pragmas := []string{
		"PRAGMA foreign_keys = ON;",
		"PRAGMA busy_timeout = 5000;",
	}
	if !isMemory {
		pragmas = append(pragmas, "PRAGMA journal_mode = WAL;", "PRAGMA synchronous = NORMAL;")
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to execute pragma '%s': %w", pragma, err)
		}
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	if err := migrateSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run database migrations: %w", err)
	}

	return db, nil
}

func migrateSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS quizzes (
		id TEXT PRIMARY KEY,
		instructor_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		title TEXT NOT NULL,
		description TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS questions (
		id TEXT PRIMARY KEY,
		quiz_id TEXT NOT NULL REFERENCES quizzes(id) ON DELETE CASCADE,
		question_text TEXT NOT NULL,
		question_type TEXT NOT NULL DEFAULT 'MULTIPLE_CHOICE',
		time_limit_sec INTEGER NOT NULL DEFAULT 30,
		points INTEGER NOT NULL DEFAULT 1000,
		order_index INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS options (
		id TEXT PRIMARY KEY,
		question_id TEXT NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
		option_text TEXT NOT NULL,
		is_correct BOOLEAN NOT NULL DEFAULT 0,
		order_index INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		quiz_id TEXT NOT NULL REFERENCES quizzes(id) ON DELETE CASCADE,
		instructor_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		pin TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'LOBBY',
		started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		ended_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS responses (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
		question_id TEXT NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
		student_id TEXT NOT NULL,
		student_nickname TEXT NOT NULL,
		selected_option_id TEXT REFERENCES options(id) ON DELETE SET NULL,
		is_correct BOOLEAN NOT NULL DEFAULT 0,
		response_time_ms INTEGER NOT NULL DEFAULT 0,
		score_awarded INTEGER NOT NULL DEFAULT 0,
		submitted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(session_id, question_id, student_id)
	);

	CREATE TABLE IF NOT EXISTS qa_questions (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
		student_id TEXT NOT NULL,
		student_nickname TEXT NOT NULL,
		content TEXT NOT NULL,
		upvotes INTEGER NOT NULL DEFAULT 0,
		is_answered BOOLEAN NOT NULL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_quizzes_instructor_id ON quizzes(instructor_id);
	CREATE INDEX IF NOT EXISTS idx_questions_quiz_id ON questions(quiz_id);
	CREATE INDEX IF NOT EXISTS idx_options_question_id ON options(question_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_pin ON sessions(pin);
	CREATE INDEX IF NOT EXISTS idx_responses_session_question ON responses(session_id, question_id);
	CREATE INDEX IF NOT EXISTS idx_qa_questions_session_id ON qa_questions(session_id);
	`

	_, err := db.Exec(schema)
	return err
}
