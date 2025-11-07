package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc"
	"main-service/internal/handlers"
	proto "posts-service/proto"
	statspb "stats-service/proto"
)

type e2ePostsClient struct {
	listResp *proto.ListPostsResponse
}

func (c *e2ePostsClient) CreatePost(context.Context, *proto.CreatePostRequest, ...grpc.CallOption) (*proto.CreatePostResponse, error) {
	return nil, nil
}

func (c *e2ePostsClient) UpdatePost(context.Context, *proto.UpdatePostRequest, ...grpc.CallOption) (*proto.UpdatePostResponse, error) {
	return nil, nil
}

func (c *e2ePostsClient) DeletePost(context.Context, *proto.DeletePostRequest, ...grpc.CallOption) (*proto.DeletePostResponse, error) {
	return nil, nil
}

func (c *e2ePostsClient) GetPost(context.Context, *proto.GetPostRequest, ...grpc.CallOption) (*proto.GetPostResponse, error) {
	return nil, nil
}

func (c *e2ePostsClient) ListPosts(context.Context, *proto.ListPostsRequest, ...grpc.CallOption) (*proto.ListPostsResponse, error) {
	return c.listResp, nil
}

type e2eStatsClient struct {
	postResp *statspb.PostStatsResponse
}

func (c e2eStatsClient) GetPostStats(context.Context, *statspb.PostStatsRequest, ...grpc.CallOption) (*statspb.PostStatsResponse, error) {
	return c.postResp, nil
}

func (e2eStatsClient) GetTopPosts(context.Context, *statspb.TopPostsRequest, ...grpc.CallOption) (*statspb.TopPostsResponse, error) {
	return nil, nil
}

func (e2eStatsClient) GetTopUsersByLikes(context.Context, *statspb.TopUsersRequest, ...grpc.CallOption) (*statspb.TopUsersResponse, error) {
	return nil, nil
}

func TestMainPostsFlow(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/posts", handlers.Posts(&e2ePostsClient{listResp: &proto.ListPostsResponse{Posts: []*proto.Post{{Id: "1", Title: "hello"}}}}))

	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/posts?page=1")
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if _, ok := body["posts"]; !ok {
		t.Fatalf("expected posts field, got %v", body)
	}
}

func TestMainStatsPostFlow(t *testing.T) {
	statsClient := e2eStatsClient{postResp: &statspb.PostStatsResponse{PostId: "1", Views: 10, Likes: 4}}

	mux := http.NewServeMux()
	mux.HandleFunc("/stats/post", handlers.StatsPost(statsClient))

	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/stats/post?id=1")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if body["views"] != float64(10) || body["likes"] != float64(4) {
		t.Fatalf("unexpected body: %+v", body)
	}
}
