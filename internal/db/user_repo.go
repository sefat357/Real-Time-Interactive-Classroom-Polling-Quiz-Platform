package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/sefat357/Real-Time-Interactive-Classroom-Polling-Quiz-Platform/internal/models"
)

var (
	// ErrNotFound indicates a requested entity was not found.
	ErrNotFound = errors.New("record not found")
	// ErrDuplicateEmail indicates an account with the given email already exists.
	ErrDuplicateEmail = errors.New("email already exists")
)

// UserRepo provides data access methods for users.
type UserRepo struct {
	db *sql.DB
}

// NewUserRepo creates a new UserRepo.
func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

// CreateUser inserts a new user record.
func (r *UserRepo) CreateUser(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, name, email, password_hash, created_at)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query, user.ID, user.Name, user.Email, user.PasswordHash, user.CreatedAt)
	if err != nil {
		if stringsContainsAny(err.Error(), "UNIQUE constraint failed", "UNIQUE constraint") {
			return ErrDuplicateEmail
		}
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// GetUserByEmail finds a user by email.
func (r *UserRepo) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, name, email, password_hash, created_at
		FROM users
		WHERE email = ?
	`
	var user models.User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return &user, nil
}

// GetUserByID finds a user by ID.
func (r *UserRepo) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	query := `
		SELECT id, name, email, password_hash, created_at
		FROM users
		WHERE id = ?
	`
	var user models.User
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}
	return &user, nil
}

func stringsContainsAny(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if len(sub) > 0 && len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
