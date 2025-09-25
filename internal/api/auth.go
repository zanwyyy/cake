package grpcapi

import (
	"context"
	"project/config"
	"project/internal/model"
	"project/internal/service"
	pb "project/pkg/pb"
)

type AuthService struct {
	pb.UnimplementedAuthServiceServer
	auth   service.AuthService
	config *config.Config
}

func (a *AuthService) GetUserID(ctx context.Context) int64 {
	if v, ok := ctx.Value(a.config.UserIDKey).(int64); ok {
		return v
	}
	return 0
}

func NewAuthService(auth service.AuthService, config *config.Config) *AuthService {
	return &AuthService{
		auth:   auth,
		config: config,
	}
}

func (s *AuthService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	in := model.LoginInput{
		Username: req.Username,
		Password: req.Password,
	}
	out, err := s.auth.Login(ctx, in)
	if err != nil {
		return nil, err
	}
	return &pb.LoginResponse{AccessToken: out.AccessToken}, nil
}

func (s *AuthService) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	in := model.LogoutInput{UserID: s.GetUserID(ctx)}
	out, err := s.auth.Logout(ctx, in)
	if err != nil {
		return nil, err
	}
	return &pb.LogoutResponse{Success: out.Success}, nil
}
