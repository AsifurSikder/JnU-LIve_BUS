package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type Database struct {
	db *sql.DB
}

type User struct {
	ID           string
	Username     string
	PasswordHash string
	Role         string
	FullName     string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewDatabase(databaseURL string) (*Database, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Database{db: db}, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) GetUserByUsernameAndRole(ctx context.Context, username, role string) (*User, error) {
	query := `
		SELECT id, username, password_hash, role, full_name, created_at, updated_at
		FROM users
		WHERE username = $1 AND role = $2
	`

	user := &User{}
	err := d.db.QueryRowContext(ctx, query, username, role).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Role,
		&user.FullName,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	return user, nil
}

func (d *Database) HealthCheck(ctx context.Context) error {
	return d.db.PingContext(ctx)
}
