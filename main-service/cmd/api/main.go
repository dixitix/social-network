package main

import (
	"database/sql"
	"fmt"
	"main-service/internal/handlers"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	proto "posts-service/proto"
	statspb "stats-service/proto"

	"github.com/segmentio/kafka-go"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://main_user:main_pass@localhost:5433/main_db?sslmode=disable"
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		panic(fmt.Errorf("db ping failed: %w", err))
	}
	fmt.Println("Connected to Postgres")

	postsAddr := os.Getenv("POSTS_SERVICE_ADDR")
	if postsAddr == "" {
		postsAddr = "localhost:50051"
	}

	grpcConn, err := grpc.NewClient(postsAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(fmt.Errorf("grpc new client failed: %w", err))
	}
	defer grpcConn.Close()

	postsClient := proto.NewPostsServiceClient(grpcConn)

	statsAddr := os.Getenv("STATS_SERVICE_ADDR")
	if statsAddr == "" {
		statsAddr = "stats-service:9090"
	}

	statsConn, err := grpc.NewClient(statsAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(fmt.Errorf("stats grpc client failed: %w", err))
	}
	defer statsConn.Close()

	statsClient := statspb.NewStatsServiceClient(statsConn)

	brokersEnv := os.Getenv("KAFKA_BROKERS")
	if brokersEnv == "" {
		brokersEnv = "localhost:29092"
	}
	brokers := []string{}
	for _, b := range strings.Split(brokersEnv, ",") {
		b = strings.TrimSpace(b)
		if b != "" {
			brokers = append(brokers, b)
		}
	}
	viewsTopic := os.Getenv("KAFKA_VIEWS_TOPIC")
	if viewsTopic == "" {
		viewsTopic = "post_views"
	}
	likesTopic := os.Getenv("KAFKA_LIKES_TOPIC")
	if likesTopic == "" {
		likesTopic = "post_likes"
	}

	var viewsWriter, likesWriter *kafka.Writer
	if len(brokers) > 0 {
		if err := ensureKafkaTopic(brokers, viewsTopic); err != nil {
			panic(fmt.Errorf("ensure kafka topic %q failed: %w", viewsTopic, err))
		}
		if err := ensureKafkaTopic(brokers, likesTopic); err != nil {
			panic(fmt.Errorf("ensure kafka topic %q failed: %w", likesTopic, err))
		}

		viewsWriter = &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Topic:                  viewsTopic,
			Balancer:               &kafka.LeastBytes{},
			AllowAutoTopicCreation: true,
		}
		defer viewsWriter.Close()

		likesWriter = &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Topic:                  likesTopic,
			Balancer:               &kafka.LeastBytes{},
			AllowAutoTopicCreation: true,
		}
		defer likesWriter.Close()
	}

	http.HandleFunc("/health", handlers.Health)
	http.HandleFunc("/auth/register", handlers.AuthRegister(db))
	http.HandleFunc("/auth/login", handlers.AuthLogin(db))
	http.HandleFunc("/users/me", handlers.UserMe(db))
	http.HandleFunc("/users/me/update", handlers.UserMeUpdate(db))
	http.HandleFunc("/posts", handlers.Posts(postsClient))
	http.HandleFunc("/posts/", handlers.PostsWithID(postsClient, viewsWriter, likesWriter))
	http.HandleFunc("/stats/post", handlers.StatsPost(statsClient))
	http.HandleFunc("/stats/top-posts", handlers.StatsTopPosts(statsClient, postsClient, db))
	http.HandleFunc("/stats/top-users", handlers.StatsTopUsers(statsClient, db))

	fmt.Println("Main server started on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

func ensureKafkaTopic(brokers []string, topic string) error {
	if topic == "" || len(brokers) == 0 {
		return nil
	}

	var lastErr error
	for attempt := 0; attempt < 8; attempt++ {
		conn, err := kafka.Dial("tcp", brokers[0])
		if err != nil {
			lastErr = err
		} else {
			controller, err := conn.Controller()
			if err != nil {
				lastErr = err
			} else {
				controllerAddr := net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port))
				controllerConn, err := kafka.Dial("tcp", controllerAddr)
				if err != nil {
					lastErr = err
				} else {
					err = controllerConn.CreateTopics(kafka.TopicConfig{
						Topic:             topic,
						NumPartitions:     1,
						ReplicationFactor: 1,
					})
					controllerConn.Close()
					if err == nil || strings.Contains(err.Error(), "already exists") {
						conn.Close()
						return nil
					}
					lastErr = err
				}
			}
			conn.Close()
		}

		wait := time.Duration(attempt+1) * time.Second
		time.Sleep(wait)
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("unknown error")
	}
	return lastErr
}
