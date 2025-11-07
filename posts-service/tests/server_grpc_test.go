package tests

import (
	"testing"

	"posts-service/internal/app"
	"posts-service/internal/db"
	pb "posts-service/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type updateStub struct{}

func (s *updateStub) Create(db.Post) (db.Post, error)                     { return db.Post{}, nil }
func (s *updateStub) Delete(string, string) error                         { return nil }
func (s *updateStub) Get(string, string) (db.Post, error)                 { return db.Post{}, nil }
func (s *updateStub) List(string, int64, int64) ([]db.Post, int64, error) { return nil, 0, nil }
func (s *updateStub) Update(string, string, string, string) (db.Post, error) {
	return db.Post{}, db.ErrNotFound
}

func TestServerUpdatePostNotFound(t *testing.T) {
	srv := &app.Server{DB: &updateStub{}}

	_, err := srv.UpdatePost(nil, &pb.UpdatePostRequest{Id: "missing", UserId: "user"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected status error, got %v", err)
	}
	if st.Code() != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", st.Code())
	}
}
