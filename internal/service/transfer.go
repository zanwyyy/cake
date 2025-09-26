package service

import (
	"context"
	"fmt"
	"project/internal/model"
	"project/internal/utils"
)

type TransferRepo interface {
	ListTransactions(ctx context.Context, from int64) ([]model.Transaction, error)
	GetBalance(ctx context.Context, userID int64) (int64, error)
	GetPassword(ctx context.Context, userID int64) (string, error)
	InsertTransaction(ctx context.Context, from, to int64, amount int64) error
}

type TransferService struct {
	repo TransferRepo
}

func NewTransferService(r TransferRepo) *TransferService {
	return &TransferService{
		repo: r,
	}
}

func (s *TransferService) ListTransactions(ctx context.Context, req model.ListTransactionsInput) (*model.ListTransactionsOutput, error) {

	if err := utils.ValidateUserID(req.UserId); err != nil {
		return nil, err
	}

	txs, err := s.repo.ListTransactions(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	out := &model.ListTransactionsOutput{}
	for _, tx := range txs {
		out.Transactions = append(out.Transactions, model.Transaction{
			ID:     tx.ID,
			From:   tx.From,
			To:     tx.To,
			Amount: tx.Amount,
		})
	}
	out.Number = int64(len(txs))
	return out, nil
}

func (s *TransferService) InsertTransaction(ctx context.Context, req model.SendMoneyInput) (*model.SendMoneyOutput, error) {
	if err := utils.ValidateUserID(req.From); err != nil {
		return &model.SendMoneyOutput{Success: false, ErrorMessage: err.Error()}, err
	}
	if err := utils.ValidateUserID(req.To); err != nil {
		return &model.SendMoneyOutput{Success: false, ErrorMessage: err.Error()}, err
	}
	if err := utils.ValidateAmount(req.Amount); err != nil {
		return &model.SendMoneyOutput{Success: false, ErrorMessage: err.Error()}, err
	}
	if req.From == req.To {
		return &model.SendMoneyOutput{Success: false, ErrorMessage: "from_user cannot equal to to_user"}, fmt.Errorf("cannot transfer to yourself")
	}

	err := s.repo.InsertTransaction(ctx, req.From, req.To, req.Amount)
	if err != nil {
		return &model.SendMoneyOutput{Success: false, ErrorMessage: err.Error()}, err
	}

	return &model.SendMoneyOutput{Success: true}, nil
}

func (s *TransferService) GetBalance(ctx context.Context, req model.GetBalanceInput) (*model.GetBalanceOutput, error) {

	if err := utils.ValidateUserID(req.UserId); err != nil {
		return nil, err
	}

	balance, err := s.repo.GetBalance(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	return &model.GetBalanceOutput{UserId: req.UserId, Balance: balance}, nil
}
