package service

import (
	"context"
	"log"
	"project/config"
	"project/internal/model"
	"project/internal/utils"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DBClient interface {
	GetPassword(ctx context.Context, userID int64) (string, error)
}

type AuthService struct {
	db     DBClient
	config *config.Config
}

func NewAuthService(config *config.Config, db DBClient) *AuthService {
	return &AuthService{
		db:     db,
		config: config,
	}
}

func (a *AuthService) GetUserID(ctx context.Context) int64 {
	if v, ok := ctx.Value(a.config.UserIDKey).(int64); ok {
		return v
	}
	return 0
}

func (a *AuthService) Login(ctx context.Context, req model.LoginInput) (*model.LoginOutput, error) {

	if err := utils.ValidateUserID(req.Username); err != nil {
		return nil, err
	}

	pass, err := a.db.GetPassword(ctx, req.Username)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid username or password")
	}

	check := utils.CheckPassword(pass, req.Password)
	if !check {
		return nil, status.Error(codes.Unauthenticated, "invalid username or password")
	}

	accessToken, err := utils.GenerateAccessToken(req.Username, a.config.JWT.AccessTokenTTL, a.config.JWT.AccessSecret)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate access token: %v", err)
	}

	refreshToken, err := utils.GenerateRefreshToken(req.Username, a.config.JWT.RefreshTokenTTL, a.config.JWT.RefreshSecret)
	if err != nil {
		return nil, status.Errorf(codes.Internal, " failed to generate refresh token: %v", err)
	}

	log.Printf("[Login] success user=%d ", req.Username)
	return &model.LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (a *AuthService) Refresh(ctx context.Context, in model.RefreshInput) (*model.RefreshOutput, error) {

	claims, err := utils.ValidateRefreshToken(in.RefreshToken, a.config.JWT.RefreshSecret)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "invalid token: %v", err)

	}
	accessToken, err := utils.GenerateAccessToken(claims.UserID, a.config.JWT.AccessTokenTTL, a.config.JWT.AccessSecret)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate access token: %v", err)
	}
	return &model.RefreshOutput{AccessToken: accessToken}, nil
}
