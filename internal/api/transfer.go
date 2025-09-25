package grpcapi

import (
	"context"
	"fmt"
	"project/config"
	"project/internal/model"
	"project/internal/repo"
	"project/internal/service"
	pb "project/pkg/pb"
)

type TransferService struct {
	pb.UnimplementedTransferServiceServer
	svc    service.TransferService
	pubsub repo.PubSubInterface
	config *config.Config
}

func NewTransferService(svc service.TransferService, pubsub repo.PubSubInterface, config *config.Config) *TransferService {
	return &TransferService{
		svc:    svc,
		pubsub: pubsub,
		config: config,
	}
}
func (s *TransferService) GetUserID(ctx context.Context) int64 {
	if v, ok := ctx.Value(s.config.UserIDKey).(int64); ok {
		return v
	}
	return 0
}
func (s *TransferService) SendMoney(ctx context.Context, req *pb.SendMoneyRequest) (*pb.SendMoneyResponse, error) {
	userId := s.GetUserID(ctx)
	in := model.SendMoneyInput{
		From:   userId,
		To:     req.To,
		Amount: req.Amount,
	}
	out, err := s.svc.InsertTransaction(ctx, in)
	if err != nil {
		return &pb.SendMoneyResponse{Success: out.Success, ErrorMessage: out.ErrorMessage}, err
	}
	msg := fmt.Sprintf(
		`{"from":"%d","to":"%d","amount":%d,"status":"success"}`,
		userId, req.To, req.Amount,
	)
	if err := s.pubsub.Publish([]byte(msg)); err != nil {
		fmt.Printf("[WARN] publish failed: %v\n", err)
	}
	return &pb.SendMoneyResponse{Success: out.Success, ErrorMessage: out.ErrorMessage}, nil
}

func (s *TransferService) ListTransactions(ctx context.Context, req *pb.ListTransactionsRequest) (*pb.ListTransactionsResponse, error) {
	userId := s.GetUserID(ctx)
	in := model.ListTransactionsInput{UserId: userId}
	out, err := s.svc.ListTransactions(ctx, in)
	if err != nil {
		return nil, err
	}

	resp := &pb.ListTransactionsResponse{Number: out.Number}
	for _, tx := range out.Transactions {
		resp.Transactions = append(resp.Transactions, &pb.Transaction{
			Id: int64(tx.ID), From: tx.From, To: tx.To, Amount: tx.Amount,
		})
	}
	return resp, nil
}
func (s *TransferService) GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {
	userId := s.GetUserID(ctx)
	in := model.GetBalanceInput{UserId: userId}
	out, err := s.svc.GetBalance(ctx, in)
	if err != nil {
		return nil, err
	}
	return &pb.GetBalanceResponse{UserId: out.UserId, Balance: out.Balance}, nil
}
