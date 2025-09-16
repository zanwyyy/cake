package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"project/config"
	"project/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TransferRepository interface {
	InsertTransaction(ctx context.Context, from, to string, amount int64) error
	ListTransactions(ctx context.Context, from string) ([]model.Transaction, error)
	GetBalance(ctx context.Context, userID string) (int64, error)
}

type GormTransferRepo struct {
	db     *gorm.DB
	pubsub PubSubInterface
}

func NewPostgresDB(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.Database.URL), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&model.User{}, &model.Transaction{}); err != nil {
		return nil, err
	}

	return db, nil
}

func NewPostgresTransferRepo(db *gorm.DB, pubsub PubSubInterface) TransferRepository {
	return &GormTransferRepo{
		db:     db,
		pubsub: pubsub,
	}
}

func (r *GormTransferRepo) ListTransactions(ctx context.Context, from string) ([]model.Transaction, error) {
	var txs []model.Transaction
	if err := r.db.WithContext(ctx).
		Where("from_user = ?", from).
		Find(&txs).Error; err != nil {
		return nil, err
	}
	return txs, nil
}

func (r *GormTransferRepo) InsertTransaction(ctx context.Context, from, to string, amount int64) error {
	e := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var fromUser, toUser model.User

		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&fromUser, "id = ?", from).Error; err != nil {
			return err
		}

		if amount < 0 {
			return fmt.Errorf("amount can't be negative")
		}

		if fromUser.Balance < amount {
			return fmt.Errorf("insufficient balance")
		}

		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&toUser, "id = ?", to).Error; err != nil {
			return err
		}

		if err := tx.Model(&fromUser).
			Update("balance", fromUser.Balance-amount).Error; err != nil {
			return err
		}
		if err := tx.Model(&toUser).
			Update("balance", toUser.Balance+amount).Error; err != nil {
			return err
		}

		newTx := model.Transaction{
			From:   from,
			To:     to,
			Amount: amount,
		}
		if err := tx.Create(&newTx).Error; err != nil {
			return err
		}

		return nil
	})
	if e == nil {
		event := model.TransactionEvent{
			From:   from,
			To:     to,
			Amount: amount,
			Status: "success",
		}
		msg, _ := json.Marshal(event)
		if err := r.pubsub.Publish(msg); err != nil {
			log.Printf("failed to publish: %v", err)
			return err
		}
	}
	return e
}

func (r *GormTransferRepo) GetBalance(ctx context.Context, userID string) (int64, error) {
	var user model.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return 0, err
	}
	return user.Balance, nil
}
