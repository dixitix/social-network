package tests

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	postspb "posts-service/proto"
	"stats-service/internal/app"
	"stats-service/internal/storage"
	statspb "stats-service/proto"
)

type likesRepoStub struct{}

func (likesRepoStub) SaveEvent(context.Context, storage.Event) error          { return nil }
func (likesRepoStub) PostStats(context.Context, string) (int64, int64, error) { return 0, 0, nil }
func (likesRepoStub) TopPosts(context.Context, string, int) ([]storage.PostCount, error) {
	return nil, nil
}
func (likesRepoStub) LikesPerPost(context.Context) ([]storage.PostCount, error) {
	return []storage.PostCount{{PostID: "p1", Value: 7}, {PostID: "p2", Value: 3}}, nil
}

type postsStub struct{}

func (postsStub) GetPost(ctx context.Context, in *postspb.GetPostRequest, opts ...grpc.CallOption) (*postspb.GetPostResponse, error) {
	return &postspb.GetPostResponse{Post: &postspb.Post{OwnerId: "user" + in.GetId(), Id: in.GetId()}}, nil
}

func TestGetTopUsersByLikes(t *testing.T) {
	srv := app.NewStatsServerForTest(likesRepoStub{}, postsStub{})

	resp, err := srv.GetTopUsersByLikes(context.Background(), &statspb.TopUsersRequest{Limit: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetUsers()) != 2 {
		t.Fatalf("expected 2 users, got %d", len(resp.GetUsers()))
	}
	if resp.GetUsers()[0].GetUserId() != "userp1" {
		t.Fatalf("unexpected first user: %+v", resp.GetUsers()[0])
	}
}
