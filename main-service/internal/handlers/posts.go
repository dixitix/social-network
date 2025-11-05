package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	proto "posts-service/proto"

	"github.com/segmentio/kafka-go"
)

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func Posts(client proto.PostsServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			page := 1
			pageSize := 10

			if v := r.URL.Query().Get("page"); v != "" {
				n, err := strconv.Atoi(v)
				if err != nil || n < 1 {
					http.Error(w, "invalid page parameter", http.StatusBadRequest)
					return
				}
				page = n
			}
			if v := r.URL.Query().Get("page_size"); v != "" {
				n, err := strconv.Atoi(v)
				if err != nil || n <= 0 {
					http.Error(w, "invalid page_size parameter", http.StatusBadRequest)
					return
				}
				pageSize = n
			}

			resp, err := client.ListPosts(context.Background(), &proto.ListPostsRequest{
				Page:     int32(page),
				PageSize: int32(pageSize),
			})
			if err != nil {
				if st, ok := status.FromError(err); ok && st.Code() == codes.InvalidArgument {
					http.Error(w, st.Message(), http.StatusBadRequest)
					return
				}
				http.Error(w, "service error", http.StatusBadGateway)
				return
			}

			respondJSON(w, http.StatusOK, resp)
		case http.MethodPost:
			AuthMiddleware(func(w http.ResponseWriter, r *http.Request, userID string) {
				var req struct {
					Title   string `json:"title"`
					Content string `json:"content"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					http.Error(w, "bad request", http.StatusBadRequest)
					return
				}

				resp, err := client.CreatePost(context.Background(), &proto.CreatePostRequest{
					UserId:  userID,
					Title:   req.Title,
					Content: req.Content,
				})
				if err != nil {
					http.Error(w, "service error", http.StatusBadGateway)
					return
				}

				respondJSON(w, http.StatusCreated, resp.Post)
			})(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func PostsWithID(client proto.PostsServiceClient, viewsWriter, likesWriter *kafka.Writer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/posts/")
		if path == "" {
			http.NotFound(w, r)
			return
		}

		parts := strings.Split(path, "/")
		id := parts[0]
		if id == "" {
			http.NotFound(w, r)
			return
		}

		action := ""
		if len(parts) > 1 {
			action = parts[1]
		}
		if len(parts) > 2 {
			http.NotFound(w, r)
			return
		}

		switch action {
		case "":
			switch r.Method {
			case http.MethodGet:
				resp, err := client.GetPost(context.Background(), &proto.GetPostRequest{Id: id})
				if err != nil {
					http.Error(w, "service error", http.StatusBadGateway)
					return
				}

				respondJSON(w, http.StatusOK, resp.Post)
			case http.MethodPut:
				AuthMiddleware(func(w http.ResponseWriter, r *http.Request, userID string) {
					var req struct {
						Title   string `json:"title"`
						Content string `json:"content"`
					}
					if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
						http.Error(w, "bad request", http.StatusBadRequest)
						return
					}

					resp, err := client.UpdatePost(context.Background(), &proto.UpdatePostRequest{
						Id:      id,
						UserId:  userID,
						Title:   req.Title,
						Content: req.Content,
					})
					if err != nil {
						http.Error(w, "service error", http.StatusBadGateway)
						return
					}

					respondJSON(w, http.StatusOK, resp.Post)
				})(w, r)
			case http.MethodDelete:
				AuthMiddleware(func(w http.ResponseWriter, r *http.Request, userID string) {
					resp, err := client.DeletePost(context.Background(), &proto.DeletePostRequest{
						Id:     id,
						UserId: userID,
					})
					if err != nil {
						http.Error(w, "service error", http.StatusBadGateway)
						return
					}

					respondJSON(w, http.StatusOK, resp)
				})(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		case "view":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			if viewsWriter == nil {
				http.Error(w, "service error", http.StatusBadGateway)
				return
			}
			if err := sendPostEvent(r.Context(), viewsWriter, id, "view"); err != nil {
				http.Error(w, "service error", http.StatusBadGateway)
				return
			}
			w.WriteHeader(http.StatusAccepted)
		case "like":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			if likesWriter == nil {
				http.Error(w, "service error", http.StatusBadGateway)
				return
			}
			if err := sendPostEvent(r.Context(), likesWriter, id, "like"); err != nil {
				http.Error(w, "service error", http.StatusBadGateway)
				return
			}
			w.WriteHeader(http.StatusAccepted)
		default:
			http.NotFound(w, r)
		}
	}
}

func sendPostEvent(ctx context.Context, writer *kafka.Writer, postID, eventType string) error {
	payload := struct {
		PostID    string    `json:"post_id"`
		EventType string    `json:"event_type"`
		Timestamp time.Time `json:"timestamp"`
	}{
		PostID:    postID,
		EventType: eventType,
		Timestamp: time.Now().UTC(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return writer.WriteMessages(ctx, kafka.Message{Value: data})
}
