package service

import (
	"context"
	"project/internal/repo"
	pb "project/pkg/pb"
)

type TransferService struct {
	repo repo.TransferRepository
}

func NewTransferService(r repo.TransferRepository) *TransferService {
	return &TransferService{repo: r}
}

func (s *TransferService) ListTransactions(ctx context.Context, req *pb.ListTransactionsRequest) (*pb.ListTransactionsResponse, error) {
	txs, err := s.repo.ListTransactions(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	resp := &pb.ListTransactionsResponse{}
	for _, tx := range txs {
		resp.Transactions = append(resp.Transactions, &pb.Transaction{
			Id:     tx.ID,
			From:   tx.From,
			To:     tx.To,
			Amount: tx.Amount,
		})
	}
	resp.Number = int64(len(txs))
	return resp, nil
}

func (s *TransferService) InsertTransaction(ctx context.Context, from, to string, amount int64) (*pb.SendMoneyResponse, error) {
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
