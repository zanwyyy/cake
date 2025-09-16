package repo

import (
	"context"
	"fmt"
	"log"
	"project/config"
	"time"

	pubsub "cloud.google.com/go/pubsub/apiv1"
	pubsubpb "cloud.google.com/go/pubsub/apiv1/pubsubpb"

	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type PubSubInterface interface {
	Subscribe(ctx context.Context) error
	Publish(data []byte) error
}

type PubSub struct {
	pubClient *pubsub.PublisherClient
	subClient *pubsub.SubscriberClient
	config    *config.Config
}

func NewPubSubClient(config *config.Config) (PubSubInterface, error) {
	ctx := context.Background()

	opts := []option.ClientOption{
		option.WithEndpoint(config.PubSub.Endpoint),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	}

	// Publisher client
	pubClient, err := pubsub.NewPublisherClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub publisher client: %w", err)
	}

	// Subscriber client
	subClient, err := pubsub.NewSubscriberClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub subscriber client: %w", err)
	}

	return &PubSub{
		pubClient: pubClient,
		subClient: subClient,
		config:    config,
	}, nil
}

func (p *PubSub) Publish(data []byte) error {
	topicPath := fmt.Sprintf("projects/%s/topics/%s",
		p.config.PubSub.ProjectID, p.config.PubSub.Topic)

	var lastErr error
	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		resp, err := p.pubClient.Publish(ctx, &pubsubpb.PublishRequest{
			Topic: topicPath,
			Messages: []*pubsubpb.PubsubMessage{
				{Data: data},
			},
		})
		if err == nil {
			fmt.Printf("[PubSub v2] Published message IDs: %v\n", resp.MessageIds)
			return nil
		}

		lastErr = err
		log.Printf("[PubSub v2] Publish attempt %d failed: %v", i+1, err)
	}
	return fmt.Errorf("failed to publish after retries: %w", lastErr)
}

func (p *PubSub) Subscribe(ctx context.Context) error {
	subPath := fmt.Sprintf("projects/%s/subscriptions/%s",
		p.config.PubSub.ProjectID, p.config.PubSub.Subcription)

	log.Println("Starting PubSub consumer...")

	for {
		resp, err := p.subClient.Pull(ctx, &pubsubpb.PullRequest{
			Subscription: subPath,
			MaxMessages:  2,
		})

		if err != nil {
			continue
		}

		if resp == nil || len(resp.ReceivedMessages) == 0 {
			continue
		}

		ackIDs := make([]string, 0, len(resp.ReceivedMessages))
		for _, m := range resp.ReceivedMessages {
			log.Printf("[PubSub v2] Received message: %s", string(m.Message.Data))
			ackIDs = append(ackIDs, m.AckId)
		}

		if err := p.subClient.Acknowledge(context.Background(), &pubsubpb.AcknowledgeRequest{
			Subscription: subPath,
			AckIds:       ackIDs,
		}); err != nil {
			log.Printf("[PubSub v2] Ack error: %v", err)
		} else {
			log.Printf("[PubSub v2] Ack success: %d message(s)", len(ackIDs))
		}

	}
}
