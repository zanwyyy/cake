package grpcapi

import (
	"context"
	"project/internal/service"
	pb "project/pkg/pb"
)

type TransferService struct {
	pb.UnimplementedTransferServiceServer
	svc service.TransferService
}

func NewTransferService(svc service.TransferService) *TransferService {
	return &TransferService{svc: svc}
}

func (s *TransferService) SendMoney(ctx context.Context, req *pb.SendMoneyRequest) (*pb.SendMoneyResponse, error) {
	return s.svc.InsertTransaction(ctx, req.From, req.To, req.Amount)
}

func (s *TransferService) ListTransactions(ctx context.Context, req *pb.ListTransactionsRequest) (*pb.ListTransactionsResponse, error) {
	return s.svc.ListTransactions(ctx, req)
}

func (s *TransferService) GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {
	return s.svc.GetBalance(ctx, req)
}
