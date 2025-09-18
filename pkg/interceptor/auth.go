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
}

func isProtectedMethod(method string) bool {
	_, ok := protectedMethods[method]
	return ok
}

func AuthInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	config := config.LoadConfig()
	redis := repo.NewRedisClient(config)

	if !isProtectedMethod(info.FullMethod) {
		return handler(ctx, req)
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "metadata not found")
	}

	// Lấy Authorization header
	authHeader := md.Get("authorization")
	if len(authHeader) == 0 {
		return nil, status.Error(codes.Unauthenticated, "authorization header missing")
	}

	// Format: Bearer <token>
	tokenString := strings.TrimPrefix(authHeader[0], "Bearer ")
	if tokenString == authHeader[0] {
		return nil, status.Error(codes.Unauthenticated, "invalid authorization header format")
	}

	// Verify JWT
	claims, err := utils.ValidateAccessToken(tokenString)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}

	// Check token trong Redis
	storedToken := redis.GetToken(ctx, claims.UserID)
	if storedToken == "" {
		return nil, status.Error(codes.Unauthenticated, "token revoked or expired")
	}

	if tokenString != storedToken {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token: ")
	}

	switch info.FullMethod {
	case "/transfer.v1.TransferService/SendMoney":
		r, ok := req.(*pb.SendMoneyRequest)
		if !ok {
			return nil, status.Error(codes.Internal, "invalid request type for SendMoney")
		}
		if r.Base.UserId != claims.UserID {
			return nil, status.Error(codes.PermissionDenied, "user id mismatch")
		}

	case "/transfer.v1.TransferService/ListTransactions":
		r, ok := req.(*pb.ListTransactionsRequest)
		if !ok {
			return nil, status.Error(codes.Internal, "invalid request type for ListTransactions")
		}
		if r.Base.UserId != claims.UserID {
			return nil, status.Error(codes.PermissionDenied, "user id mismatch")
		}

	case "/transfer.v1.TransferService/GetBalance":
		r, ok := req.(*pb.GetBalanceRequest)
		if !ok {
			return nil, status.Error(codes.Internal, "invalid request type for GetBalance")
		}
		if r.Base.UserId != claims.UserID {
			return nil, status.Error(codes.PermissionDenied, "user id mismatch")
		}
	}
	return handler(ctx, req)

}
