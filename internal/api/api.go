package grpcapi

import (
	"context"
	"project/internal/service"
	pb "project/pkg/pb"
)

type TransferService struct {
	pb.UnimplementedTransferServiceServer
	svc  service.TransferService
	auth service.AuthService
}

func NewTransferService(svc service.TransferService, auth service.AuthService) *TransferService {
	return &TransferService{
		svc:  svc,
		auth: auth,
	}
}

func (s *TransferService) SendMoney(ctx context.Context, req *pb.SendMoneyRequest) (*pb.SendMoneyResponse, error) {
	return s.svc.InsertTransaction(ctx, req.Base.UserId, req.To, req.Amount)
}

func (s *TransferService) ListTransactions(ctx context.Context, req *pb.ListTransactionsRequest) (*pb.ListTransactionsResponse, error) {
	return s.svc.ListTransactions(ctx, req)
}

func (s *TransferService) GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {
	return s.svc.GetBalance(ctx, req)
}

func (s *TransferService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	return s.auth.Login(ctx, req)
}

func (s *TransferService) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	return s.auth.Logout(ctx, req)
}
