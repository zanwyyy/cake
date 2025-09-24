package service

import (
	"context"
	"fmt"
	"project/internal/repo"
	"project/internal/utils"
	pb "project/pkg/pb"
)

type TransferService interface {
	ListTransactions(ctx context.Context, req *pb.ListTransactionsRequest) (*pb.ListTransactionsResponse, error)
	InsertTransaction(ctx context.Context, from, to int64, amount int64) (*pb.SendMoneyResponse, error)
	GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error)
}

type transferService struct {
	repo   repo.TransferRepository
	pubsub repo.PubSubInterface
}

func NewTransferService(r repo.TransferRepository, pb repo.PubSubInterface) TransferService {
	return &transferService{
		repo:   r,
		pubsub: pb,
	}
}

func (s *transferService) ListTransactions(ctx context.Context, req *pb.ListTransactionsRequest) (*pb.ListTransactionsResponse, error) {

	if err := utils.ValidateUserID(req.UserId); err != nil {
		return nil, err
	}

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

	if err := utils.ValidateUserID(from); err != nil {
		return &pb.SendMoneyResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, err
	}
	if err := utils.ValidateUserID(to); err != nil {
		return &pb.SendMoneyResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, err
	}

	if err := utils.ValidateAmount(amount); err != nil {
		return &pb.SendMoneyResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, err
	}

	if from == to {
		return &pb.SendMoneyResponse{
			Success:      false,
			ErrorMessage: "from_user can equal to to_user",
		}, fmt.Errorf("cannot transfer to yourself")
	}

	err := s.repo.InsertTransaction(ctx, from, to, amount)
	if err != nil {
		return &pb.SendMoneyResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, err
	}
	msg := fmt.Sprintf(
		`{"from":"%d","to":"%d","amount":%d,"status":"success"}`,
		from, to, amount,
	)
	s.pubsub.Publish([]byte(msg))
	return &pb.SendMoneyResponse{
		Success:      true,
		ErrorMessage: "",
	}, nil
}

func (s *transferService) GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {

	if err := utils.ValidateUserID(req.UserId); err != nil {
		return nil, err
	}

	balance, err := s.repo.GetBalance(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	return &pb.GetBalanceResponse{
		UserId:  req.UserId,
		Balance: balance,
	}, nil
}
