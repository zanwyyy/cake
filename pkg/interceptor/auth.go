package interceptor

import (
	"context"
	"strings"

	"project/config"
	"project/internal/repo"
	"project/internal/utils"
	"project/pkg/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var protectedMethods = map[string]struct{}{
	"/transfer.v1.TransferService/SendMoney":        {},
	"/transfer.v1.TransferService/ListTransactions": {},
	"/transfer.v1.TransferService/GetBalance":       {},
	"/transfer.v1.AuthService/Logout":               {},
}

func isProtectedMethod(method string) bool {
	_, ok := protectedMethods[method]
	return ok
}

func NewAuthInterceptor(redis repo.RedisClient, config *config.Config) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {

		if !isProtectedMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "metadata not found")
		}

		authHeader := md.Get("authorization")
		if len(authHeader) == 0 {
			return nil, status.Error(codes.Unauthenticated, "authorization header missing")
		}

		tokenString := strings.TrimPrefix(authHeader[0], "Bearer ")
		if tokenString == authHeader[0] {
			return nil, status.Error(codes.Unauthenticated, "invalid authorization header format")
		}

		claims, err := utils.ValidateAccessToken(tokenString, config.JWT.AccessSecret)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		}

		var userID int64
		switch r := req.(type) {
		case *pb.SendMoneyRequest:
			userID = r.From
		case *pb.GetBalanceRequest:
			userID = r.UserId
		case *pb.ListTransactionsRequest:
			userID = r.UserId
		default:
			userID = claims.UserID
		}

		storedToken := redis.GetToken(ctx, userID)
		if storedToken == "" {
			return nil, status.Error(codes.Unauthenticated, "token revoked or expired")
		}
		if storedToken != tokenString {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}
		ctx = context.WithValue(ctx, config.UserIDKey, userID)
		return handler(ctx, req)
	}
}
