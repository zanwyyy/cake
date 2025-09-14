package repo

import (
	"context"
	"log"
	"project/config"
	"time"

	"github.com/segmentio/kafka-go"
)

type Kafka interface {
	Publish(ctx context.Context, key, value string) error
}

type KafkaWriter struct {
	writer *kafka.Writer
}

type KafkaConsumer struct {
	reader *kafka.Reader
	ps     *PubSub
}

func NewKafkaConsumer(ps *PubSub, config *config.Config) *KafkaConsumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{config.Kafka.Broker},
		Topic:          config.Kafka.Topic,
		GroupID:        "demo-consumer-group1",
		CommitInterval: 0 * time.Second,
		StartOffset:    kafka.FirstOffset,
	})
	return &KafkaConsumer{
		reader: r,
		ps:     ps,
	}

}

func NewKafkaWriter(config *config.Config) Kafka {
	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  []string{config.Kafka.Broker},
		Topic:    config.Kafka.Topic,
		Balancer: &kafka.LeastBytes{},
	})
	return &KafkaWriter{writer: w}
}

func (kw *KafkaWriter) Publish(ctx context.Context, key, value string) error {
	err := kw.writer.WriteMessages(ctx,
		kafka.Message{
			Key:   []byte(key),
			Value: []byte(value),
		},
	)
	if err != nil {
		log.Printf("failed to write message: %v", err)
		return err
	}
	log.Printf("Published to Kafka: key=%s value=%s", key, value)
	return nil
}

func (c *KafkaConsumer) Consume(ctx context.Context) {
	log.Println("Kafka consumer started...")
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("error reading kafka message: %v", err)

			if err == context.DeadlineExceeded || err == context.Canceled {
				log.Println("Context timeout/cancelled, retrying...")
				time.Sleep(2 * time.Second)
				continue
			}

			log.Printf("Kafka error, retrying in 5 seconds: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}
		log.Printf("Kafka msg: topic=%s key=%s value=%s",
			m.Topic, string(m.Key), string(m.Value))

		if err := c.ps.Publish(m.Value); err != nil {
			log.Printf("failed to publish to PubSub: %v", err)

			time.Sleep(2 * time.Second)
			continue
		}
	}

}
