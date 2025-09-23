package repo

import (
	"context"
	"fmt"
	"project/config"
	"project/internal/model"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

type TransferRepository interface {
	InsertTransaction(ctx context.Context, from, to int64, amount int64) error
	ListTransactions(ctx context.Context, from int64) ([]model.Transaction, error)
	GetBalance(ctx context.Context, userID int64) (int64, error)
	GetPassword(ctx context.Context, userId int64) (string, error)
}

type GormTransferRepo struct {
	db *gorm.DB
}

func NewPostgresDB(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open("postgres", cfg.Database.URL)
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&model.User{}, &model.Transaction{}).Error; err != nil {
		return nil, err
	}

	return db, nil
}

func NewPostgresTransferRepo(db *gorm.DB) TransferRepository {
	return &GormTransferRepo{
		db: db,
	}
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

	tx := r.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	var firstID, secondID int64
	if from < to {
		firstID, secondID = from, to
	} else {
		firstID, secondID = to, from
	}

	var user1, user2 model.User

	if err := model.NewUserQuerySet(tx.Set("gorm:query_option", "FOR UPDATE")).
		IDEq(firstID).
		One(&user1); err != nil {
		tx.Rollback()
		if gorm.IsRecordNotFoundError(err) {
			return fmt.Errorf("user %d not found", firstID)
		}
		return err
	}

	if err := model.NewUserQuerySet(tx.Set("gorm:query_option", "FOR UPDATE")).
		IDEq(secondID).
		One(&user2); err != nil {
		tx.Rollback()
		if gorm.IsRecordNotFoundError(err) {
			return fmt.Errorf("user %d not found", secondID)
		}
		return err
	}

	var fromUser, toUser *model.User
	if firstID == from {
		fromUser = &user1
		toUser = &user2
	} else {
		fromUser = &user2
		toUser = &user1
	}

	if fromUser.Balance < amount {
		tx.Rollback()
		return fmt.Errorf("insufficient balance")
	}

	if err := tx.Model(fromUser).Update("balance", fromUser.Balance-amount).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(toUser).Update("balance", toUser.Balance+amount).Error; err != nil {
		tx.Rollback()
		return err
	}

	newTx := model.Transaction{
		From:   from,
		To:     to,
		Amount: amount,
	}
	if err := tx.Create(&newTx).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (r *GormTransferRepo) GetBalance(ctx context.Context, userID int64) (int64, error) {
	var user model.User
	err := model.NewUserQuerySet(r.db).
		IDEq(userID).
		Limit(1).
		One(&user)

	if err != nil {
		return 0, err
	}
	return user.Balance, nil
}

func (r *GormTransferRepo) GetPassword(ctx context.Context, userID int64) (string, error) {
	var user model.User
	err := model.NewUserQuerySet(r.db).
		IDEq(userID).
		Limit(1).
		One(&user)

	if err != nil {
		return "", err
	}
	return user.Password, nil
}
