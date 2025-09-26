package grpcapi

import (
	"context"
	"fmt"
	"project/config"
	"project/internal/model"
	pb "project/pkg/pb"
)

type Publisher interface {
	Publish(data []byte) error
}

type TransferService interface {
	ListTransactions(ctx context.Context, in model.ListTransactionsInput) (*model.ListTransactionsOutput, error)
	InsertTransaction(ctx context.Context, in model.SendMoneyInput) (*model.SendMoneyOutput, error)
	GetBalance(ctx context.Context, in model.GetBalanceInput) (*model.GetBalanceOutput, error)
}

type Transfer struct {
	pb.UnimplementedTransferServiceServer
	svc    TransferService
	pubsub Publisher
	config *config.Config
}

func NewTransferService(svc TransferService, pubsub Publisher, config *config.Config) *Transfer {
	return &Transfer{
		svc:    svc,
		pubsub: pubsub,
		config: config,
	}
}
func (s *Transfer) GetUserID(ctx context.Context) int64 {
	if v, ok := ctx.Value(s.config.UserIDKey).(int64); ok {
		return v
	}
	return 0
}
func (s *Transfer) SendMoney(ctx context.Context, req *pb.SendMoneyRequest) (*pb.SendMoneyResponse, error) {
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

func (s *Transfer) ListTransactions(ctx context.Context, req *pb.ListTransactionsRequest) (*pb.ListTransactionsResponse, error) {
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
func (s *Transfer) GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {
	userId := s.GetUserID(ctx)
	in := model.GetBalanceInput{UserId: userId}
	out, err := s.svc.GetBalance(ctx, in)
	if err != nil {
		return nil, err
	}
	return &pb.GetBalanceResponse{UserId: out.UserId, Balance: out.Balance}, nil
}
