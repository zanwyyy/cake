package grpcapi

import (
	"context"
	"project/internal/service"
	pb "project/pkg/pb"
)

type AuthService struct {
	pb.UnimplementedAuthServiceServer
	auth service.AuthService
}

func NewAuthService(auth service.AuthService) *AuthService {
	return &AuthService{
		auth: auth,
	}
}

func (s *AuthService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	return s.auth.Login(ctx, req)
}

func (s *AuthService) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	return s.auth.Logout(ctx, req)
}
