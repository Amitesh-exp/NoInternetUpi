package bank

import (
	"database/sql"
	"errors"
)

type Account struct {
	UPI     string
	Balance float64
}

type AccountStore struct {
	db *sql.DB
}

// NewAccountStore opens the SQLite database and creates
// the accounts table if it doesn't already exist
func NewAccountStore(db *sql.DB) (*AccountStore, error) {
	createTable := `
	CREATE TABLE IF NOT EXISTS accounts (
		upi     TEXT PRIMARY KEY,
		balance REAL NOT NULL
	);`

	_, err := db.Exec(createTable)
	if err != nil {
		return nil, err
	}

	return &AccountStore{db: db}, nil
}

// Seed adds initial accounts if they don't exist yet
// This is how we set up test accounts with starting balances
func (s *AccountStore) Seed() error {
	accounts := []Account{
		{"alice@upi", 1000.00},
		{"bob@upi", 500.00},
		{"charlie@upi", 750.00},
	}

	for _, acc := range accounts {
		_, err := s.db.Exec(`
			INSERT OR IGNORE INTO accounts (upi, balance) VALUES (?, ?)`,
			acc.UPI, acc.Balance,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetBalance returns current balance for a UPI ID
func (s *AccountStore) GetBalance(upi string) (float64, error) {
	var balance float64
	err := s.db.QueryRow(`
		SELECT balance FROM accounts WHERE upi = ?`, upi,
	).Scan(&balance)

	if err == sql.ErrNoRows {
		return 0, errors.New("account not found: " + upi)
	}
	return balance, err
}

// Transfer moves amount from sender to receiver atomically
// Atomic means both deduction and addition happen together
// or neither happens — no partial transfers ever
func (s *AccountStore) Transfer(senderUPI string, receiverUPI string, amount float64) error {

	// Begin a transaction — this is what makes it atomic
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	// If anything goes wrong below, undo everything
	defer tx.Rollback()

	// Check sender balance
	var senderBalance float64
	err = tx.QueryRow(`
		SELECT balance FROM accounts WHERE upi = ?`, senderUPI,
	).Scan(&senderBalance)

	if err == sql.ErrNoRows {
		return errors.New("sender account not found")
	}
	if err != nil {
		return err
	}

	if senderBalance < amount {
		return errors.New("insufficient balance")
	}

	// Deduct from sender
	_, err = tx.Exec(`
		UPDATE accounts SET balance = balance - ? WHERE upi = ?`,
		amount, senderUPI,
	)
	if err != nil {
		return err
	}

	// Add to receiver
	_, err = tx.Exec(`
		UPDATE accounts SET balance = balance + ? WHERE upi = ?`,
		amount, receiverUPI,
	)
	if err != nil {
		return err
	}

	// Commit — only now do the changes actually save
	return tx.Commit()
}