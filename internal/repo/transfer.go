package repo

import (
	"context"
	"errors"
	"fmt"
	"project/config"
	"project/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GormTransferRepo struct {
	db *gorm.DB
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

func NewPostgresTransferRepo(db *gorm.DB) *GormTransferRepo {
	return &GormTransferRepo{db: db}
}

func (r *GormTransferRepo) ListTransactions(ctx context.Context, from int64) ([]model.Transaction, error) {
	var txs []model.Transaction
	err := model.NewTransactionQuerySet(r.db).
		FromEq(from).
		All(&txs)
	if err != nil {
		return nil, err
	}
	return txs, nil
}

func (r *GormTransferRepo) InsertTransaction(ctx context.Context, from, to int64, amount int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var firstID, secondID int64
		if from < to {
			firstID, secondID = from, to
		} else {
			firstID, secondID = to, from
		}

		var user1, user2 model.User

		if err := model.NewUserQuerySet(tx.Clauses(clause.Locking{Strength: "UPDATE"})).
			IDEq(firstID).
			One(&user1); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("user %d not found", firstID)
			}
			return err
		}

		if err := model.NewUserQuerySet(tx.Clauses(clause.Locking{Strength: "UPDATE"})).
			IDEq(secondID).
			One(&user2); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("user %d not found", secondID)
			}
			return err
		}

		var fromUser, toUser *model.User
		if firstID == from {
			fromUser, toUser = &user1, &user2
		} else {
			fromUser, toUser = &user2, &user1
		}

		if fromUser.Balance < amount {
			return fmt.Errorf("insufficient balance")
		}

		if err := model.NewUserQuerySet(tx).
			IDEq(fromUser.ID).
			GetUpdater().
			SetBalance(fromUser.Balance - amount).
			Update(); err != nil {
			return err
		}

		if err := model.NewUserQuerySet(tx).
			IDEq(toUser.ID).
			GetUpdater().
			SetBalance(toUser.Balance + amount).
			Update(); err != nil {
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
}

func (r *GormTransferRepo) GetBalance(ctx context.Context, userID int64) (int64, error) {
	var user model.User
	err := model.NewUserQuerySet(r.db).IDEq(userID).One(&user)
	if err != nil {
		return 0, err
	}
	return user.Balance, nil
}

func (r *GormTransferRepo) GetPassword(ctx context.Context, userID int64) (string, error) {
	var user model.User
	err := model.NewUserQuerySet(r.db).IDEq(userID).One(&user)
	if err != nil {
		return "", err
	}
	return user.Password, nil
}
