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

func (s *AccountStore) Transfer(senderUPI string, receiverUPI string, amount float64) error {

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

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

	_, err = tx.Exec(`
		UPDATE accounts SET balance = balance - ? WHERE upi = ?`,
		amount, senderUPI,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		UPDATE accounts SET balance = balance + ? WHERE upi = ?`,
		amount, receiverUPI,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}