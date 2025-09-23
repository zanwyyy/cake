package interceptor

import (
	"context"
	"fmt"
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

func isProtectedMethod(method string) bool {
	switch method {
	case
		"/transfer.v1.TransferService/SendMoney",
		"/transfer.v1.TransferService/ListTransactions",
		"/transfer.v1.TransferService/GetBalance",
		"/transfer.v1.AuthService/Logout":
		return true
	default:
		return false
	}
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
			return nil, status.Errorf(codes.Unauthenticated, "validate : invalid token: %v", err)
		}

		if claims.UserID <= 0 {
			return nil, status.Error(codes.Unauthenticated, "invalid user id in token")
		}

		storedToken, err := redis.GetToken(ctx, claims.UserID)
		if err != nil {
			return nil, status.Error(codes.Internal, "auth service unavailable")
		}
		if storedToken == "" {
			return nil, status.Error(codes.Unauthenticated, "token revoked or expired")
		}
		if strings.TrimSpace(storedToken) != strings.TrimSpace(tokenString) {
			return nil, status.Error(codes.Unauthenticated, "token mismatch")
		}

		fmt.Println(claims.UserID)

		switch r := req.(type) {
		case *pb.SendMoneyRequest:
			if r.From != claims.UserID {
				fmt.Println(r.From)
				return nil, status.Error(codes.PermissionDenied, "you can only send from your own account")
			}
		case *pb.GetBalanceRequest:
			if r.UserId != claims.UserID {
				fmt.Print(r.UserId)
				return nil, status.Error(codes.PermissionDenied, "access denied")
			}
		case *pb.ListTransactionsRequest:
			if r.UserId != claims.UserID {
				fmt.Print(r.UserId)
				return nil, status.Error(codes.PermissionDenied, "access denied")
			}
		}

		ctx = context.WithValue(ctx, config.UserIDKey, claims.UserID)
		return handler(ctx, req)
	}
}
