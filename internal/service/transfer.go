package service

import (
	"context"
	"project/internal/repo"
	pb "project/pkg/pb"
)

type TransferService interface {
	ListTransactions(ctx context.Context, req *pb.ListTransactionsRequest) (*pb.ListTransactionsResponse, error)
	InsertTransaction(ctx context.Context, from, to int64, amount int64) (*pb.SendMoneyResponse, error)
	GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error)
}

type transferService struct {
	repo repo.TransferRepository
}

func NewTransferService(r repo.TransferRepository) TransferService {
	return &transferService{repo: r}
}

func (s *transferService) ListTransactions(ctx context.Context, req *pb.ListTransactionsRequest) (*pb.ListTransactionsResponse, error) {
	txs, err := s.repo.ListTransactions(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	resp := &pb.ListTransactionsResponse{}
	for _, tx := range txs {
		resp.Transactions = append(resp.Transactions, &pb.Transaction{
			Id:     int64(tx.ID),
			From:   tx.From,
			To:     tx.To,
			Amount: tx.Amount,
		})
	}
	resp.Number = int64(len(txs))
	return resp, nil
}

func (s *transferService) InsertTransaction(ctx context.Context, from, to int64, amount int64) (*pb.SendMoneyResponse, error) {
	err := s.repo.InsertTransaction(ctx, from, to, amount)
	if err != nil {
		return &pb.SendMoneyResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, err
	}
	return &pb.SendMoneyResponse{
		Success:      true,
		ErrorMessage: "",
	}, nil
}

func (s *transferService) GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {
	balance, err := s.repo.GetBalance(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	return &pb.GetBalanceResponse{
		UserId:  req.UserId,
		Balance: balance,
	}, nil
}
