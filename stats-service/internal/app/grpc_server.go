package app

import (
	"context"
	"sort"

	"google.golang.org/grpc"
	postspb "posts-service/proto"
	statspb "stats-service/proto"
)

type postsClient interface {
	GetPost(ctx context.Context, in *postspb.GetPostRequest, opts ...grpc.CallOption) (*postspb.GetPostResponse, error)
}

type statsServer struct {
	statspb.UnimplementedStatsServiceServer
	repo        statsRepository
	postsClient postsClient
}

func newStatsServer(repo statsRepository, postsClient postsClient) *statsServer {
	return &statsServer{repo: repo, postsClient: postsClient}
}

// NewStatsServerForTest exposes a stats server with injected dependencies for tests.
func NewStatsServerForTest(repo statsRepository, postsClient postsClient) statspb.StatsServiceServer {
	return newStatsServer(repo, postsClient)
}

func (s *statsServer) GetPostStats(ctx context.Context, in *statspb.PostStatsRequest) (*statspb.PostStatsResponse, error) {
	if in == nil || in.GetPostId() == "" {
		return &statspb.PostStatsResponse{}, nil
	}
	views, likes, err := s.repo.PostStats(ctx, in.GetPostId())
	if err != nil {
		return nil, err
	}
	return &statspb.PostStatsResponse{PostId: in.GetPostId(), Views: views, Likes: likes}, nil
}

func (s *statsServer) GetTopPosts(ctx context.Context, in *statspb.TopPostsRequest) (*statspb.TopPostsResponse, error) {
	metric := in.GetMetric()
	if metric == "" {
		metric = "views"
	}
	eventType := "view"
	if metric == "likes" {
		eventType = "like"
	} else if metric != "views" {
		metric = "views"
	}
	limit := int(in.GetLimit())
	if limit <= 0 {
		limit = 5
	}

	posts, err := s.repo.TopPosts(ctx, eventType, limit)
	if err != nil {
		return nil, err
	}

	resp := &statspb.TopPostsResponse{}
	for _, p := range posts {
		resp.Items = append(resp.Items, &statspb.PostItem{PostId: p.PostID, Value: p.Value})
	}
	return resp, nil
}

func (s *statsServer) GetTopUsersByLikes(ctx context.Context, in *statspb.TopUsersRequest) (*statspb.TopUsersResponse, error) {
	limit := int(in.GetLimit())
	if limit <= 0 {
		limit = 3
	}

	posts, err := s.repo.LikesPerPost(ctx)
	if err != nil {
		return nil, err
	}

	userLikes := make(map[string]int64)
	for _, p := range posts {
		ownerID, err := s.postOwner(ctx, p.PostID)
		if err != nil || ownerID == "" {
			continue
		}
		userLikes[ownerID] += p.Value
	}

	type pair struct {
		userID string
		likes  int64
	}
	var sorted []pair
	for userID, likes := range userLikes {
		if likes > 0 {
			sorted = append(sorted, pair{userID: userID, likes: likes})
		}
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].likes > sorted[j].likes })
	if len(sorted) > limit {
		sorted = sorted[:limit]
	}

	resp := &statspb.TopUsersResponse{}
	for _, item := range sorted {
		resp.Users = append(resp.Users, &statspb.UserItem{UserId: item.userID, Likes: item.likes})
	}
	return resp, nil
}

func (s *statsServer) postOwner(ctx context.Context, postID string) (string, error) {
	if s.postsClient == nil {
		return "", nil
	}
	resp, err := s.postsClient.GetPost(ctx, &postspb.GetPostRequest{Id: postID})
	if err != nil {
		return "", err
	}
	if resp == nil || resp.Post == nil {
		return "", nil
	}
	return resp.Post.GetOwnerId(), nil
}
