package repo

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"project/config"
	"project/internal/model"

	_ "github.com/lib/pq"
)

type TransferRepository interface {
	InsertTransaction(ctx context.Context, from, to string, amount int64) error
	ListTransactions(ctx context.Context, from string) ([]model.Transaction, error)
}

type PostgresTransferRepo struct {
	db    *sql.DB
	kafka Kafka
}

func NewPostgresDB(config *config.Config) (*sql.DB, error) {
	dsn := config.Database.URL
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	fmt.Println(db.Ping())
	return db, db.Ping()
}

func NewPostgresTransferRepo(db *sql.DB, kakfa Kafka) TransferRepository {
	return &PostgresTransferRepo{
		db:    db,
		kafka: kakfa,
	}
}

func (r *PostgresTransferRepo) ListTransactions(ctx context.Context, from string) ([]model.Transaction, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, from_user, to_user, amount
         FROM transactions
         WHERE from_user = $1
         `, from,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []model.Transaction
	for rows.Next() {
		var tx model.Transaction
		if err := rows.Scan(&tx.ID, &tx.From, &tx.To, &tx.Amount); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, nil
}
func (r *PostgresTransferRepo) InsertTransaction(ctx context.Context, from, to string, amount int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		}
	}()

	var balance int64
	err = tx.QueryRowContext(ctx,
		`SELECT balance FROM users WHERE id = $1 FOR UPDATE`,
		from,
	).Scan(&balance)
	if err != nil {
		return err
	}
	fmt.Println(balance)
	if balance < amount {
		return fmt.Errorf("insufficient balance")
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE users SET balance = balance - $1 WHERE id = $2`,
		amount, from,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE users SET balance = balance + $1 WHERE id = $2`,
		amount, to,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO transactions (from_user, to_user, amount) VALUES ($1, $2, $3)`,
		from, to, amount,
	)
	if err != nil {
		return err
	}

	err = tx.Commit()
	msg := fmt.Sprintf("Transaction from %s to %s with amount %d success", from, to, amount)
	if err = r.kafka.Publish(ctx, from, msg); err != nil {
		log.Printf("failed to publish: %v", err)
	}
	return err
}
