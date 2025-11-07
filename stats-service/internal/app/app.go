package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	postspb "posts-service/proto"
	statspb "stats-service/proto"

	"stats-service/internal/storage"
)

type Config struct {
	HTTPAddr           string
	GRPCAddr           string
	ClickHouseAddr     []string
	ClickHouseDB       string
	ClickHouseUser     string
	ClickHousePassword string
	KafkaBrokers       []string
	KafkaGroupID       string
	ViewsTopic         string
	LikesTopic         string
	PostsServiceAddr   string
}

type event struct {
	PostID    string    `json:"post_id"`
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
}

type kafkaMessageReader interface {
	ReadMessage(context.Context) (kafka.Message, error)
	Close() error
}

var newKafkaReader = func(cfg kafka.ReaderConfig) kafkaMessageReader {
	return kafka.NewReader(cfg)
}

type statsRepository interface {
	SaveEvent(ctx context.Context, e storage.Event) error
	PostStats(ctx context.Context, postID string) (int64, int64, error)
	TopPosts(ctx context.Context, eventType string, limit int) ([]storage.PostCount, error)
	LikesPerPost(ctx context.Context) ([]storage.PostCount, error)
}

func Run(ctx context.Context, cfg Config) error {
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = ":8081"
	}
	if cfg.GRPCAddr == "" {
		cfg.GRPCAddr = ":9090"
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
	if cfg.PostsServiceAddr == "" {
		cfg.PostsServiceAddr = "posts-service:50051"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: mux,
	}
	errCh := make(chan error, 2)

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http server failed: %w", err)
		}
	}()

	grpcListener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("grpc listen failed: %w", err)
	}

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

	postsConn, err := grpc.NewClient(cfg.PostsServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		grpcListener.Close()
		return fmt.Errorf("posts client init failed: %w", err)
	}
	defer postsConn.Close()

	grpcSrv := grpc.NewServer()
	statspb.RegisterStatsServiceServer(grpcSrv, newStatsServer(repo, postspb.NewPostsServiceClient(postsConn)))

	go func() {
		if err := grpcSrv.Serve(grpcListener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			errCh <- fmt.Errorf("grpc server failed: %w", err)
		}
	}()

	go func() {
		<-ctx.Done()
		grpcSrv.GracefulStop()
	}()

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

func consumeTopic(ctx context.Context, wg *sync.WaitGroup, repo statsRepository, cfg kafka.ReaderConfig, defaultType string) {
	defer wg.Done()

	reader := newKafkaReader(cfg)
	defer reader.Close()

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
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
