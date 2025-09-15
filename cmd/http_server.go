package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	pb "project/pkg/pb"

	"project/config"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/fx"
	"google.golang.org/grpc"
)

type HTTPGateway struct {
	Mux      *runtime.ServeMux
	HTTPAddr string
	GRPCAddr string
}

func NewHTTPGateway(config *config.Config) *HTTPGateway {
	fmt.Println(config.Gateway.GRPCAddr)
	return &HTTPGateway{
		Mux:      runtime.NewServeMux(),
		HTTPAddr: config.Gateway.HTTPAddr,
		GRPCAddr: config.Gateway.GRPCAddr,
	}
}

func RegisterHTTPLifecycle(lc fx.Lifecycle, gw *HTTPGateway) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				gatewayCtx := context.Background()
				opts := []grpc.DialOption{grpc.WithInsecure()}

				if err := pb.RegisterTransferServiceHandlerFromEndpoint(
					gatewayCtx, gw.Mux, gw.GRPCAddr, opts,
				); err != nil {
					log.Println(err)
				}

				log.Printf("HTTP Gateway listening on %s (proxy to %s)", gw.HTTPAddr, gw.GRPCAddr)
				if err := http.ListenAndServe(gw.HTTPAddr, gw.Mux); err != nil {
					log.Println(err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			// Ở đây có thể graceful shutdown nếu cần
			return nil
		},
	})
}
