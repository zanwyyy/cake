package cmd

import (
	"context"
	"fmt"
	"log"
	"net"

	"go.uber.org/fx"
	"google.golang.org/grpc"

	"project/config"
	grpcapi "project/internal/api"
	"project/internal/service"
	pb "project/pkg/pb"
)

type GRPCServer struct {
	*grpc.Server
	Addr string
}

func NewGRPCServer(svc service.TransferService, config *config.Config) *GRPCServer {
	s := grpc.NewServer()
	pb.RegisterTransferServiceServer(s, grpcapi.NewTransferService(svc))
	return &GRPCServer{
		Server: s,
		Addr:   config.Server.GRPCAddr,
	}
}

func RegisterGRPCLifecycle(lc fx.Lifecycle, srv *GRPCServer) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				lis, err := net.Listen("tcp", srv.Addr)
				if err != nil {
					panic(err)
				}
				fmt.Println("gRPC server listening on ", srv.Addr)
				if err := srv.Serve(lis); err != nil && err != grpc.ErrServerStopped {
					log.Fatalf("failed to serve: %v", err)
				}

			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Println("stopping gRPC server...")
			srv.GracefulStop()
			return nil
		},
	})
}
