package storage

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
)

type Storage struct {
	conn *sql.DB
}

type Scanner interface {
	Scan(dest ...interface{}) error
}

func NewStorage(databaseURL string) (*Storage, error) {
	conn, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("could not open sql: %w", err)
	}

	return &Storage{
		conn: conn,
	}, nil
}

func (s *Storage) StoreRefreshToken(ctx context.Context, userID, hashedToken string) error {
	_, err := s.conn.ExecContext(ctx, "INSERT INTO refresh_tokens(user_id, token) VALUES($1, $2)", userID, hashedToken)
	return err
}

func (s *Storage) GetRefreshToken(ctx context.Context, userID string) (string, error) {
	var hashedToken string
	err := s.conn.QueryRowContext(ctx, "SELECT token FROM refresh_tokens WHERE user_id = $1", userID).Scan(&hashedToken)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return hashedToken, nil
}
