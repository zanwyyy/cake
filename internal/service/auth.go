package service

import (
	"context"
	"log"
	"project/config"
	"project/internal/repo"
	"project/internal/utils"
	"project/pkg/pb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthService interface {
	Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error)
	Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error)
}

type authService struct {
	db     repo.TransferRepository
	config *config.Config
	redis  repo.RedisClient
}

func NewauthService(config *config.Config, db repo.TransferRepository, redis repo.RedisClient) AuthService {
	return &authService{
		db:     db,
		config: config,
		redis:  redis,
	}
}

func (a *authService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	pass, err := a.db.GetPassword(ctx, req.Username)
	if err != nil || pass != req.Password {
		return nil, status.Error(codes.Unauthenticated, "invalid username or password")
	}

	tokenID := utils.NewSessionID()

	accessToken, err := utils.GenerateAccessToken(req.Username, tokenID, a.config.JWT.AccessTokenTTL, a.config.JWT.AccessSecret)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate access token: %v", err)
	}

	a.redis.SaveToken(ctx, req.Username, accessToken, a.config.JWT.AccessTokenTTL)

	resp := &pb.LoginResponse{
		AccessToken: accessToken,
	}

	log.Printf("[Login] success user=%d tokenID=%s", req.Username, tokenID)
	return resp, nil
}

func (a *authService) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {

	err := a.redis.DeleteToken(ctx, req.UserId)
	if err != nil {
		log.Printf("[Logout] failed to remove token for user=%d: %v", req.UserId, err)
		return nil, status.Error(codes.Internal, "failed to logout")
	}

	log.Printf("[Logout] user=%d logged out successfully", req.UserId)
	return &pb.LogoutResponse{Success: true}, nil
}
