package tests

import (
	"errors"
	"testing"

	"posts-service/internal/app"
	"posts-service/internal/db"
	pb "posts-service/proto"
)

type stubPostStore struct {
	createFn func(db.Post) (db.Post, error)
}

func (s *stubPostStore) Create(p db.Post) (db.Post, error) {
	if s.createFn != nil {
		return s.createFn(p)
	}
	return db.Post{}, errors.New("not implemented")
}

func (s *stubPostStore) Update(string, string, string, string) (db.Post, error) {
	return db.Post{}, errors.New("not implemented")
}

func (s *stubPostStore) Delete(string, string) error { return errors.New("not implemented") }

func (s *stubPostStore) Get(string, string) (db.Post, error) {
	return db.Post{}, errors.New("not implemented")
}

func (s *stubPostStore) List(string, int64, int64) ([]db.Post, int64, error) {
	return nil, 0, errors.New("not implemented")
}

func TestToPBConversion(t *testing.T) {
	post := db.Post{ID: "id1", OwnerID: "user1", Title: "hello", Content: "world"}
	pbPost := app.ToPBForTest(post)

	if pbPost.GetId() != post.ID {
		t.Fatalf("expected id %q, got %q", post.ID, pbPost.GetId())
	}
	if pbPost.GetOwnerId() != post.OwnerID {
		t.Fatalf("expected owner %q, got %q", post.OwnerID, pbPost.GetOwnerId())
	}
	if pbPost.GetTitle() != post.Title || pbPost.GetContent() != post.Content {
		t.Fatalf("unexpected post content: %+v", pbPost)
	}
}

func TestServerCreatePost(t *testing.T) {
	var saved db.Post
	store := &stubPostStore{createFn: func(p db.Post) (db.Post, error) {
		saved = p
		p.ID = "generated"
		return p, nil
	}}

	srv := &app.Server{DB: store}

	resp, err := srv.CreatePost(nil, &pb.CreatePostRequest{UserId: "user42", Title: "t", Content: "c"})
	if err != nil {
		t.Fatalf("CreatePost returned error: %v", err)
	}
	if saved.OwnerID != "user42" || saved.Title != "t" || saved.Content != "c" {
		t.Fatalf("unexpected saved post: %+v", saved)
	}
	if resp.GetPost().GetId() != "generated" {
		t.Fatalf("expected generated id, got %q", resp.GetPost().GetId())
	}
}
