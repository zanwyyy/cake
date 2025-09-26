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
					fx.Annotate(
						repo.NewPostgresTransferRepo,
						fx.As(new(service.TransferRepo)),
						fx.As(new(service.DBClient)),
					),
					fx.Annotate(
						service.NewTransferService,
						fx.As(new(grpcapi.TransferService)),
					),
					fx.Annotate(
						repo.NewPubSubClient,
						fx.As(new(grpcapi.Publisher)),
					),
					config.LoadConfig,
					service.NewAuthService,

					fx.Annotate(
						repo.NewRedisClient,
						fx.As(new(service.RedisClient)),
						fx.As(new(interceptor.RedisToken)),
					),
					interceptor.NewAuthInterceptor,
					fx.Annotate(
						service.NewAuthService,
						fx.As(new(grpcapi.AuthService)),
					),
					grpcapi.NewTransferService,
					grpcapi.NewAuth,
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
