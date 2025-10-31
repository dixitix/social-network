package app

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"posts-service/internal/db"
	pb "posts-service/proto"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	pb.UnimplementedPostsServiceServer
	DB *db.DB
}

func toPB(p db.Post) *pb.Post {
	return &pb.Post{
		Id:        p.ID,
		OwnerId:   p.OwnerID,
		Title:     p.Title,
		Content:   p.Content,
		CreatedAt: timestamppb.New(p.CreatedAt),
		UpdatedAt: timestamppb.New(p.UpdatedAt),
	}
}

func (s *Server) CreatePost(_ context.Context, in *pb.CreatePostRequest) (*pb.CreatePostResponse, error) {
	p, err := s.DB.Create(db.Post{
		ID:      db.NewStringID(),
		OwnerID: in.GetUserId(),
		Title:   in.GetTitle(),
		Content: in.GetContent(),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.CreatePostResponse{Post: toPB(p)}, nil
}

func (s *Server) UpdatePost(_ context.Context, in *pb.UpdatePostRequest) (*pb.UpdatePostResponse, error) {
	p, err := s.DB.Update(in.GetId(), in.GetUserId(), in.GetTitle(), in.GetContent())
	if err != nil {
		if err == db.ErrNotFound {
			return nil, status.Error(codes.NotFound, "not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.UpdatePostResponse{Post: toPB(p)}, nil
}

func (s *Server) DeletePost(_ context.Context, in *pb.DeletePostRequest) (*pb.DeletePostResponse, error) {
	err := s.DB.Delete(in.GetId(), in.GetUserId())
	if err != nil {
		if err == db.ErrNotFound {
			return nil, status.Error(codes.NotFound, "not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.DeletePostResponse{Success: true}, nil
}

func (s *Server) GetPost(_ context.Context, in *pb.GetPostRequest) (*pb.GetPostResponse, error) {
	p, err := s.DB.Get(in.GetId(), in.GetUserId())
	if err != nil {
		if err == db.ErrNotFound {
			return nil, status.Error(codes.NotFound, "not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.GetPostResponse{Post: toPB(p)}, nil
}

func (s *Server) ListPosts(_ context.Context, in *pb.ListPostsRequest) (*pb.ListPostsResponse, error) {
	page := in.GetPage()
	pageSize := in.GetPageSize()

	posts, total, err := s.DB.List(in.GetUserId(), int64(page), int64(pageSize))
	if err != nil {
		if errors.Is(err, db.ErrInvalidPagination) {
			return nil, status.Error(codes.InvalidArgument, "invalid pagination parameters")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	var out []*pb.Post
	for _, p := range posts {
		out = append(out, toPB(p))
	}
	return &pb.ListPostsResponse{
		Posts:    out,
		Page:     page,
		PageSize: pageSize,
		Total:    int32(total),
	}, nil
}
