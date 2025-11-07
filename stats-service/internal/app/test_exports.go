package app

import (
	"context"
	"sync"

	"github.com/segmentio/kafka-go"
)

// StatsRepositoryForTest exposes the internal repository interface for external tests.
type StatsRepositoryForTest = statsRepository

// KafkaMessageReaderForTest exposes the kafka reader interface for external tests.
type KafkaMessageReaderForTest = kafkaMessageReader

// SetKafkaReaderFactoryForTest replaces the kafka reader factory for the duration of a test.
func SetKafkaReaderFactoryForTest(factory func(kafka.ReaderConfig) kafkaMessageReader) (restore func()) {
	original := newKafkaReader
	newKafkaReader = factory
	return func() { newKafkaReader = original }
}

// ConsumeTopicForTest calls the internal consumeTopic helper for integration tests.
func ConsumeTopicForTest(ctx context.Context, wg *sync.WaitGroup, repo statsRepository, cfg kafka.ReaderConfig, defaultEvent string) {
	consumeTopic(ctx, wg, repo, cfg, defaultEvent)
}
