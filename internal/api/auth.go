package grpcapi

import (
	"context"
	"project/config"
	"project/internal/model"
	pb "project/pkg/pb"
)

type AuthService interface {
	Login(ctx context.Context, in model.LoginInput) (*model.LoginOutput, error)
	Logout(ctx context.Context, in model.LogoutInput) (*model.LogoutOutput, error)
}

type Auth struct {
	pb.UnimplementedAuthServiceServer
	auth   AuthService
	config *config.Config
}

func (a *Auth) GetUserID(ctx context.Context) int64 {
	if v, ok := ctx.Value(a.config.UserIDKey).(int64); ok {
		return v
	}
	return 0
}

func NewAuth(auth AuthService, config *config.Config) *Auth {
	return &Auth{
		auth:   auth,
		config: config,
	}
}

func (s *Auth) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
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

func (s *Auth) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	in := model.LogoutInput{UserID: s.GetUserID(ctx)}
	out, err := s.auth.Logout(ctx, in)
	if err != nil {
		return nil, err
	}
	return &pb.LogoutResponse{Success: out.Success}, nil
}
