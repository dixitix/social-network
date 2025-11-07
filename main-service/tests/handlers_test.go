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
)

type listPostsClient struct {
	resp *proto.ListPostsResponse
	err  error
}

func (c *listPostsClient) CreatePost(context.Context, *proto.CreatePostRequest, ...grpc.CallOption) (*proto.CreatePostResponse, error) {
	return nil, nil
}

func (c *listPostsClient) UpdatePost(context.Context, *proto.UpdatePostRequest, ...grpc.CallOption) (*proto.UpdatePostResponse, error) {
	return nil, nil
}

func (c *listPostsClient) DeletePost(context.Context, *proto.DeletePostRequest, ...grpc.CallOption) (*proto.DeletePostResponse, error) {
	return nil, nil
}

func (c *listPostsClient) GetPost(context.Context, *proto.GetPostRequest, ...grpc.CallOption) (*proto.GetPostResponse, error) {
	return nil, nil
}

func (c *listPostsClient) ListPosts(context.Context, *proto.ListPostsRequest, ...grpc.CallOption) (*proto.ListPostsResponse, error) {
	return c.resp, c.err
}

func TestPostsHandlerRespondsJSON(t *testing.T) {
	handler := handlers.Posts(&listPostsClient{resp: &proto.ListPostsResponse{Posts: []*proto.Post{{Id: "1", Title: "hello"}}}})

	req := httptest.NewRequest(http.MethodGet, "/posts?page=1", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected json content type, got %q", ct)
	}
	var body proto.ListPostsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if len(body.Posts) != 1 || body.Posts[0].GetId() != "1" {
		t.Fatalf("unexpected response body: %+v", body.Posts)
	}
}

func TestPostsHandlerInvalidPage(t *testing.T) {
	handler := handlers.Posts(&listPostsClient{})

	req := httptest.NewRequest(http.MethodGet, "/posts?page=abc", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}
