package repo

import (
	"context"
	"fmt"
	"log"
	"project/config"
	"project/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TransferRepository interface {
	InsertTransaction(ctx context.Context, from, to int64, amount int64) error
	ListTransactions(ctx context.Context, from int64) ([]model.Transaction, error)
	GetBalance(ctx context.Context, userID int64) (int64, error)
	GetPassword(ctx context.Context, userId int64) (string, error)
}

type GormTransferRepo struct {
	db     *gorm.DB
	pubsub PubSubInterface
}

func validateUserID(ctx context.Context, r *GormTransferRepo, userID int64) error {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID).
		Count(&count).Error

	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("user not exist")
	}

	return nil
}

func validateAmount(amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("amount cannot be negative or zero")
	}

	const maxAmount = 1000000000
	if amount > maxAmount {
		return fmt.Errorf("amount exceeds maximum limit")
	}

	return nil
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

func (r *GormTransferRepo) ListTransactions(ctx context.Context, from int64) ([]model.Transaction, error) {
	var txs []model.Transaction

	if err := r.db.WithContext(ctx).
		Where("from_user = ?", from).
		Find(&txs).Error; err != nil {
		return nil, err
	}
	return txs, nil
}

func (r *GormTransferRepo) InsertTransaction(ctx context.Context, from, to int64, amount int64) error {
	if err := validateAmount(amount); err != nil {
		return err
	}

	if from == to {
		return fmt.Errorf("from_user can't be equal to to_user")
	}
	e := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var fromUser, toUser model.User

		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&fromUser, "id = ?", from).Error; err != nil {
			return err
		}
		if fromUser.Balance < amount {
			return fmt.Errorf("balance < amount")
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

		msg := fmt.Sprintf(
			`{"from":"%d","to":"%d","amount":%d,"status":"success"}`,
			from, to, amount,
		)

		if err := r.pubsub.Publish([]byte(msg)); err != nil {
			log.Printf("failed to publish: %v", err)
			return err
		}
		return nil
	})

	return e
}

func (r *GormTransferRepo) GetBalance(ctx context.Context, userID int64) (int64, error) {
	var balance int64

	if err := r.db.WithContext(ctx).
		Table("users").
		Select("balance").
		Where("id = ?", userID).
		Scan(&balance).Error; err != nil {
		return 0, err
	}
	return balance, nil
}

func (r *GormTransferRepo) GetPassword(ctx context.Context, userID int64) (string, error) {
	var password string

	if err := r.db.WithContext(ctx).
		Table("users").
		Select("password").
		Where("id = ?", userID).
		Scan(&password).Error; err != nil {
		return "", err
	}

	return password, nil
}
