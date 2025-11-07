package tests

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"

	"stats-service/internal/app"
	"stats-service/internal/storage"
)

type memoryRepo struct {
	mu     sync.Mutex
	events []storage.Event
}

func (m *memoryRepo) SaveEvent(_ context.Context, e storage.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, e)
	return nil
}

func (m *memoryRepo) PostStats(_ context.Context, postID string) (int64, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var views, likes int64
	for _, e := range m.events {
		if e.PostID != postID {
			continue
		}
		if e.EventType == "view" {
			views++
		}
		if e.EventType == "like" {
			likes++
		}
	}
	return views, likes, nil
}

func (m *memoryRepo) TopPosts(context.Context, string, int) ([]storage.PostCount, error) {
	return nil, nil
}
func (m *memoryRepo) LikesPerPost(context.Context) ([]storage.PostCount, error) { return nil, nil }

type stubReader struct {
	messages []kafka.Message
	cancel   context.CancelFunc
}

func (s *stubReader) ReadMessage(ctx context.Context) (kafka.Message, error) {
	if len(s.messages) == 0 {
		if s.cancel != nil {
			s.cancel()
		}
		return kafka.Message{}, context.Canceled
	}
	msg := s.messages[0]
	s.messages = s.messages[1:]
	return msg, nil
}

func (s *stubReader) Close() error { return nil }

func TestConsumeTopicSavesEvents(t *testing.T) {
	repo := &memoryRepo{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	message := map[string]any{"post_id": "p1", "event_type": "view", "timestamp": time.Now()}
	value, _ := json.Marshal(message)
	second := map[string]any{"post_id": "p1"}
	value2, _ := json.Marshal(second)

	reader := &stubReader{messages: []kafka.Message{{Value: value}, {Value: value2}}, cancel: cancel}
	restore := app.SetKafkaReaderFactoryForTest(func(kafka.ReaderConfig) app.KafkaMessageReaderForTest { return reader })
	defer restore()

	var wg sync.WaitGroup
	wg.Add(1)

	go app.ConsumeTopicForTest(ctx, &wg, repo, kafka.ReaderConfig{}, "like")

	wg.Wait()

	views, likes, err := repo.PostStats(context.Background(), "p1")
	if err != nil {
		t.Fatalf("post stats error: %v", err)
	}
	if views != 1 || likes != 1 {
		t.Fatalf("expected 1 view and 1 like, got %d and %d", views, likes)
	}
}
