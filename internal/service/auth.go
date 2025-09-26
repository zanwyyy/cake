package service

import (
	"context"
	"log"
	"project/config"
	"project/internal/model"
	"project/internal/utils"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DBClient interface {
	GetPassword(ctx context.Context, userID int64) (string, error)
}

type RedisClient interface {
	SaveToken(ctx context.Context, userID int64, token string, ttl time.Duration) error
	DeleteToken(ctx context.Context, userID int64) error
}
type AuthService struct {
	db     DBClient
	config *config.Config
	redis  RedisClient
}

func NewAuthService(config *config.Config, db DBClient, redis RedisClient) *AuthService {
	return &AuthService{
		db:     db,
		config: config,
		redis:  redis,
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

	if err := a.redis.SaveToken(ctx, req.Username, accessToken, a.config.JWT.AccessTokenTTL); err != nil {
		log.Printf("[Login] Redis save token failed: %v", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	log.Printf("[Login] success user=%d ", req.Username)
	return &model.LoginOutput{AccessToken: accessToken}, nil
}

func (a *AuthService) Logout(ctx context.Context, in model.LogoutInput) (*model.LogoutOutput, error) {
	userID := a.GetUserID(ctx)
	err := a.redis.DeleteToken(ctx, userID)
	if err != nil {
		log.Printf("[Logout] failed to remove token for user=%d: %v", userID, err)
		return nil, status.Error(codes.Internal, "failed to logout")
	}

	log.Printf("[Logout] user=%d logged out successfully", userID)
	return &model.LogoutOutput{Success: true}, nil
}
