package service

import (
	"context"
	"fmt"
	"project/internal/model"
	"project/internal/repo"
	"project/internal/utils"
)

type TransferService interface {
	ListTransactions(ctx context.Context, in model.ListTransactionsInput) (*model.ListTransactionsOutput, error)
	InsertTransaction(ctx context.Context, in model.SendMoneyInput) (*model.SendMoneyOutput, error)
	GetBalance(ctx context.Context, in model.GetBalanceInput) (*model.GetBalanceOutput, error)
}

type transferService struct {
	repo repo.TransferRepository
}

func NewTransferService(r repo.TransferRepository) TransferService {
	return &transferService{
		repo: r,
	}
}

func (s *transferService) ListTransactions(ctx context.Context, req model.ListTransactionsInput) (*model.ListTransactionsOutput, error) {

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

func (s *transferService) InsertTransaction(ctx context.Context, req model.SendMoneyInput) (*model.SendMoneyOutput, error) {
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

func (s *transferService) GetBalance(ctx context.Context, req model.GetBalanceInput) (*model.GetBalanceOutput, error) {

	if err := utils.ValidateUserID(req.UserId); err != nil {
		return nil, err
	}

	balance, err := s.repo.GetBalance(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	return &model.GetBalanceOutput{UserId: req.UserId, Balance: balance}, nil
}
