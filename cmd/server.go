package cmd

import (
	"project/config"
	grpcapi "project/internal/api"
	"project/internal/repo"
	"project/internal/service"
	"project/pkg/interceptor"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

func NewServeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "server",
		Short: "Start HTTP + gRPC servers",
		Run: func(cmd *cobra.Command, args []string) {
			app := fx.New(
				fx.Provide(
					NewHTTPGateway,
					NewGRPCServer,
					repo.NewPostgresDB,
					repo.NewPostgresTransferRepo,
					service.NewTransferService,
					repo.NewPubSubClient,
					config.LoadConfig,
					service.NewauthService,
					repo.NewRedisClient,
					interceptor.NewAuthInterceptor,
					grpcapi.NewAuthService,
					grpcapi.NewTransferService,
				),
				fx.Invoke(
					RegisterHTTPLifecycle,
					RegisterGRPCLifecycle,
				),
			)
			app.Run()
		},
	}
}
