package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"

	"stats-service/internal/storage"
)

type Config struct {
	HTTPAddr           string
	ClickHouseAddr     []string
	ClickHouseDB       string
	ClickHouseUser     string
	ClickHousePassword string
	KafkaBrokers       []string
	KafkaGroupID       string
	ViewsTopic         string
	LikesTopic         string
}

type event struct {
	PostID    string    `json:"post_id"`
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
}

func Run(ctx context.Context, cfg Config) error {
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = ":8081"
	}
	if len(cfg.ClickHouseAddr) == 0 {
		cfg.ClickHouseAddr = []string{"stats-clickhouse:9000"}
	}
	if cfg.ClickHouseDB == "" {
		cfg.ClickHouseDB = "stats"
	}
	if cfg.ClickHouseUser == "" {
		cfg.ClickHouseUser = "default"
	}
	if cfg.KafkaGroupID == "" {
		cfg.KafkaGroupID = "stats-service"
	}
	if cfg.ViewsTopic == "" {
		cfg.ViewsTopic = "post_views"
	}
	if cfg.LikesTopic == "" {
		cfg.LikesTopic = "post_likes"
	}
	if len(cfg.KafkaBrokers) == 0 {
		return fmt.Errorf("no kafka brokers configured")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: mux,
	}
	errCh := make(chan error, 1)

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http server failed: %w", err)
		}
	}()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("http shutdown failed: %v", err)
		}
	}()

	repo, err := storage.New(ctx, storage.Config{
		Addr:     cfg.ClickHouseAddr,
		DB:       cfg.ClickHouseDB,
		User:     cfg.ClickHouseUser,
		Password: cfg.ClickHousePassword,
	})
	if err != nil {
		return fmt.Errorf("storage init failed: %w", err)
	}
	defer repo.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	go consumeTopic(ctx, &wg, repo, kafka.ReaderConfig{
		Brokers: cfg.KafkaBrokers,
		Topic:   cfg.ViewsTopic,
		GroupID: cfg.KafkaGroupID,
	}, "view")

	go consumeTopic(ctx, &wg, repo, kafka.ReaderConfig{
		Brokers: cfg.KafkaBrokers,
		Topic:   cfg.LikesTopic,
		GroupID: cfg.KafkaGroupID,
	}, "like")

	wgDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(wgDone)
	}()

	for {
		select {
		case err := <-errCh:
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		case <-wgDone:
			return nil
		}
	}
}

func consumeTopic(ctx context.Context, wg *sync.WaitGroup, repo *storage.Repository, cfg kafka.ReaderConfig, defaultType string) {
	defer wg.Done()

	reader := kafka.NewReader(cfg)
	defer reader.Close()

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("read kafka message failed: %v", err)
			time.Sleep(time.Second)
			continue
		}

		var e event
		if err := json.Unmarshal(msg.Value, &e); err != nil {
			log.Printf("decode message failed: %v", err)
			continue
		}
		if e.EventType == "" {
			e.EventType = defaultType
		}
		if e.PostID == "" {
			log.Printf("skip message without post_id")
			continue
		}
		if e.Timestamp.IsZero() {
			e.Timestamp = time.Now().UTC()
		}

		if err := repo.SaveEvent(ctx, storage.Event{
			EventType: e.EventType,
			PostID:    e.PostID,
			Timestamp: e.Timestamp,
		}); err != nil {
			log.Printf("save event failed: %v", err)
		}
	}
}
