package config

import (
	"log"
	"os"
	"time"
)

type ctxKeyID string

type Config struct {
	Server    ServerConfig
	Gateway   GatewayConfig
	Database  DatabaseConfig
	PubSub    PubSubConfig
	JWT       JWT
	Redis     RedisConfig
	UserIDKey ctxKeyID
}

type ServerConfig struct {
	GRPCAddr string
}

type GatewayConfig struct {
	HTTPAddr string
	GRPCAddr string
}

type DatabaseConfig struct {
	URL string
}

type PubSubConfig struct {
	ProjectID   string
	Endpoint    string
	Subcription string
	Topic       string
}

type RedisConfig struct {
	RedisAddr string
	Password  string
	Db        string
}

type JWT struct {
	AccessSecret   string
	AccessTokenTTL time.Duration
}

func LoadConfig() *Config {
	cfg := &Config{
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", "postgres://demo_user:demo_pass@localhost:5432/demo_db?sslmode=disable"),
		},
		PubSub: PubSubConfig{
			ProjectID:   getEnv("PROJECT_ID", "demo-project"),
			Endpoint:    getEnv("Pubsub_Endpoint", "localhost:8085"),
			Subcription: getEnv("Pubsub_Subcription", "sub-transactions"),
			Topic:       getEnv("Pubsub_Topic", "transactions"),
		},
		Server: ServerConfig{
			GRPCAddr: getEnv("GRPC_ADDR", ":9090"),
		},
		Gateway: GatewayConfig{
			HTTPAddr: getEnv("HTTP_ADDR", ":8080"),
			GRPCAddr: getEnv("GATEWAY_GRPC_ADDR", "localhost:9090"),
		},
		JWT: JWT{
			AccessSecret:   getEnv("AccessSecret", "access"),
			AccessTokenTTL: 5 * time.Minute,
		},
		Redis: RedisConfig{
			RedisAddr: getEnv("Redis_Addr", "localhost:6379"),
			Password:  getEnv("Redis_Password", ""),
		},
		UserIDKey: ctxKeyID("userID"),
	}

	log.Printf("Loaded config: %+v\n", cfg)
	return cfg
}

func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}
