package repo

import (
	"context"
	"fmt"
	"log"
	"project/config"
	"project/internal/model"
	"unicode/utf8"

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

func validateUserID(ctx context.Context, r *GormTransferRepo, userID string) error {
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

	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	if len(userID) > 50 {
		return fmt.Errorf("user ID too long")
	}

	if !utf8.ValidString(userID) {
		return fmt.Errorf("user ID contains invalid characters")
	}

	return nil
}

func validateAmount(amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("amount cannot be negative")
	}

	const maxAmount = 1000000000 // 1 billion limit
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

func (r *GormTransferRepo) ListTransactions(ctx context.Context, from string) ([]model.Transaction, error) {
	var txs []model.Transaction
	if err := validateUserID(ctx, r, from); err != nil {
		return nil, err
	}
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

		if err := validateAmount(amount); err != nil {
			return err
		}

		if err := validateUserID(ctx, r, from); err != nil {
			return err
		}

		if err := validateUserID(ctx, r, to); err != nil {
			return err
		}

		if from == to {
			return fmt.Errorf("from_user can't be equal to to_user")
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

		return nil
	})
	if e == nil {
		msg := fmt.Sprintf(
			`{"from":"%s","to":"%s","amount":%d,"status":"success"}`,
			from, to, amount,
		)

		if err := r.pubsub.Publish([]byte(msg)); err != nil {
			log.Printf("failed to publish: %v", err)
			return err
		}
	}

	return e
}

func (r *GormTransferRepo) GetBalance(ctx context.Context, userID string) (int64, error) {
	var balance int64

	if err := validateUserID(ctx, r, userID); err != nil {
		return 0, err
	}

	if err := r.db.WithContext(ctx).
		Table("users").
		Select("balance").
		Where("id = ?", userID).
		Scan(&balance).Error; err != nil {
		return 0, err
	}
	return balance, nil
}
