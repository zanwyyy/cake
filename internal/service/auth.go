package service

import (
	"context"
	"fmt"
	"project/config"
	"project/internal/repo"
	"project/internal/utils"
	"project/pkg/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthService interface {
	Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error)
}

type authService struct {
	db     repo.TransferRepository
	config *config.Config
}

func NewauthService(config *config.Config, db repo.TransferRepository) AuthService {
	return &authService{
		db:     db,
		config: config,
	}
}

func (a *authService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {

	pass, err := a.db.GetPassword(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("invalid username or password")
	}
	if pass != req.Password {
		return nil, fmt.Errorf("invalid username or password")
	}

	// 2. Generate tokens
	accessToken, err := utils.GenerateAccessToken(req.Username)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate access token")
	}

	refreshToken, err := utils.GenerateRefreshToken(req.Username)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate refresh token")
	}

	cookie := fmt.Sprintf(
		"access_token=%s; Path=/; HttpOnly; Secure; SameSite=Strict; Max-Age=%d",
		accessToken, 15*60, // 15 minutes
	)
	md := metadata.Pairs("set-cookie", cookie)
	if err := grpc.SetHeader(ctx, md); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to set cookie")
	}

	return &pb.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
