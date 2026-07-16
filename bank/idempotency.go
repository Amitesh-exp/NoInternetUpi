package bank

import (
	"database/sql"
	"time"
)

type IdempotencyStore struct {
	db *sql.DB
}

// NewIdempotencyStore creates the processed_transactions table
// if it doesn't already exist
func NewIdempotencyStore(db *sql.DB) (*IdempotencyStore, error) {
	createTable := `
	CREATE TABLE IF NOT EXISTS processed_transactions (
		transaction_id TEXT PRIMARY KEY,
		processed_at   DATETIME NOT NULL
	);`

	_, err := db.Exec(createTable)
	if err != nil {
		return nil, err
	}

	return &IdempotencyStore{db: db}, nil
}

// HasBeenProcessed checks if this transaction ID already exists
func (s *IdempotencyStore) HasBeenProcessed(transactionID string) (bool, error) {
	var id string
	err := s.db.QueryRow(`
		SELECT transaction_id FROM processed_transactions
		WHERE transaction_id = ?`, transactionID,
	).Scan(&id)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// MarkAsProcessed records this transaction ID so duplicates are rejected
func (s *IdempotencyStore) MarkAsProcessed(transactionID string) error {
	_, err := s.db.Exec(`
		INSERT OR IGNORE INTO processed_transactions (transaction_id, processed_at)
		VALUES (?, ?)`,
		transactionID, time.Now(),
	)
	return err
}