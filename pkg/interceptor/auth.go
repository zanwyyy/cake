package interceptor

import (
	"context"
	"strings"

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

	claims, err := utils.ValidateAccessToken(tokenString)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}

	userID := claims.UserID

	if r, ok := req.(*pb.SendMoneyRequest); ok {
		if r.From != userID {
			return nil, status.Error(codes.PermissionDenied, "cannot send money from another account")
		}
	}
	return handler(ctx, req)

}
