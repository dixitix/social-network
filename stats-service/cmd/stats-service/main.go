package main

import (
	"context"
	"log"
	"os"
	"strings"

	"stats-service/internal/app"
)

func main() {
	ctx := context.Background()

	cfg := app.Config{
		HTTPAddr:           ":8081",
		ClickHouseAddr:     []string{env("CLICKHOUSE_ADDR", "stats-clickhouse:9000")},
		ClickHouseDB:       env("CLICKHOUSE_DB", "stats"),
		ClickHouseUser:     env("CLICKHOUSE_USER", "default"),
		ClickHousePassword: os.Getenv("CLICKHOUSE_PASSWORD"),
		KafkaBrokers:       splitAndClean(env("KAFKA_BROKERS", "kafka:9092")),
		KafkaGroupID:       "stats-service",
		ViewsTopic:         env("KAFKA_VIEWS_TOPIC", "post_views"),
		LikesTopic:         env("KAFKA_LIKES_TOPIC", "post_likes"),
	}

	if err := app.Run(ctx, cfg); err != nil {
		log.Fatalf("stats-service failed: %v", err)
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func splitAndClean(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
