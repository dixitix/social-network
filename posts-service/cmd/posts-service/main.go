package main

import (
	"log"
	"net"
	"os"
	"posts-service/internal/db"
	"posts-service/internal/app"

	"google.golang.org/grpc"

	pb "posts-service/proto"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	addr := env("GRPC_ADDR", ":50051")
	mongoURI := env("MONGO_URI", "mongodb://posts-mongo:27017")
	dbName := env("MONGO_DB", "postsdb")
	collName := env("MONGO_COLL", "posts")

	lis, err := net.Listen("tcp", addr)
	if err != nil { 
		log.Fatal(err) 
	}

	client, err := mongo.Connect(nil, options.Client().ApplyURI(mongoURI))
	if err != nil { 
		log.Fatal(err) 
	}

	db := db.New(client, dbName, collName)

	s := grpc.NewServer()
	pb.RegisterPostsServiceServer(s, &app.Server{DB: db})

	log.Printf("posts-service gRPC listening on %s", addr)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
