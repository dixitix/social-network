package tests

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
	postspb "posts-service/proto"
	"stats-service/internal/app"
	"stats-service/internal/storage"
	statspb "stats-service/proto"
)

type repoStub struct {
	lastEventType string
	topResp       []storage.PostCount
}

func (r *repoStub) SaveEvent(context.Context, storage.Event) error { return nil }

func (r *repoStub) PostStats(context.Context, string) (int64, int64, error) { return 3, 2, nil }

func (r *repoStub) TopPosts(ctx context.Context, eventType string, limit int) ([]storage.PostCount, error) {
	r.lastEventType = eventType
	if len(r.topResp) == 0 {
		return []storage.PostCount{{PostID: "p1", Value: 5}}, nil
	}
	if limit <= 0 {
		return nil, errors.New("invalid limit")
	}
	return r.topResp, nil
}

func (r *repoStub) LikesPerPost(context.Context) ([]storage.PostCount, error) { return nil, nil }

type postClientStub struct{}

func (postClientStub) GetPost(context.Context, *postspb.GetPostRequest, ...grpc.CallOption) (*postspb.GetPostResponse, error) {
	return &postspb.GetPostResponse{Post: &postspb.Post{OwnerId: "owner"}}, nil
}

func TestGetPostStatsNilRequest(t *testing.T) {
	srv := app.NewStatsServerForTest(&repoStub{}, nil)

	resp, err := srv.GetPostStats(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetPostId() != "" || resp.GetViews() != 0 || resp.GetLikes() != 0 {
		t.Fatalf("expected empty response, got %+v", resp)
	}
}

func TestGetTopPostsDefaultsToViews(t *testing.T) {
	repo := &repoStub{}
	srv := app.NewStatsServerForTest(repo, postClientStub{})

	resp, err := srv.GetTopPosts(context.Background(), &statspb.TopPostsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastEventType != "view" {
		t.Fatalf("expected event type view, got %q", repo.lastEventType)
	}
	if len(resp.GetItems()) != 1 || resp.GetItems()[0].GetPostId() != "p1" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}
