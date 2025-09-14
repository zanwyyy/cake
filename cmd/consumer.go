package cmd

import (
	"context"
	"fmt"
	"log"
	"project/config"
	"project/internal/repo"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

func NewConsumerCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "consumer",
		Short: "Start Kafka consumer",
		Run: func(cmd *cobra.Command, args []string) {
			app := fx.New(
				fx.Provide(
					repo.NewKafkaConsumer,
					repo.NewPubSubClient,
					config.LoadConfig,
				),
				fx.Invoke(
					RegisterConsumerLifecycle,
				),
			)
			app.Run()
		},
	}
}

func NewPubSubConsumerCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "pubsub-consumer",
		Short: "Start Google Pub/Sub consumer",
		Run: func(cmd *cobra.Command, args []string) {
			app := fx.New(
				fx.Provide(
					config.LoadConfig,
					repo.NewPubSubClient,
				),
				fx.Invoke(RegisterPubSubConsumer),
			)
			app.Run()
		},
	}
}

func RegisterConsumerLifecycle(lc fx.Lifecycle, c *repo.KafkaConsumer) {
	ctx, cancel := context.WithCancel(context.Background())

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go c.Consume(ctx)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Println("Kafka consumer stopping...")
			cancel()
			return nil
		},
	})
}

func RegisterPubSubConsumer(lc fx.Lifecycle, ps *repo.PubSub) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			fmt.Println("Starting PubSub consumer...")
			go func() {
				if err := ps.Subscribe(ctx); err != nil {
					fmt.Printf("PubSub consumer error: %v\n", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			fmt.Println("Stopping PubSub consumer...")
			// ctx cancel sẽ dừng Receive()
			return nil
		},
	})
}
