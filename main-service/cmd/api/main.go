package main

import (
	"database/sql"
	"fmt"
	"main-service/internal/handlers"
	"net/http"
	"os"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	proto "posts-service/proto"

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
		postsAddr = "localhost:9090"
	}

	grpcConn, err := grpc.NewClient(postsAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(fmt.Errorf("grpc new client failed: %w", err))
	}
	defer grpcConn.Close()

	postsClient := proto.NewPostsServiceClient(grpcConn)

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
		viewsWriter = &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    viewsTopic,
			Balancer: &kafka.LeastBytes{},
		}
		defer viewsWriter.Close()

		likesWriter = &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    likesTopic,
			Balancer: &kafka.LeastBytes{},
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

	fmt.Println("Main server started on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
