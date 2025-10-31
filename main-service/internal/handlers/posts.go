package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	proto "posts-service/proto"
)

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func Posts(client proto.PostsServiceClient) http.HandlerFunc {
	return AuthMiddleware(func(w http.ResponseWriter, r *http.Request, userID string) {
		switch r.Method {
		case http.MethodPost:
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
		case http.MethodGet:
			page := 1
			pageSize := 10

			if v := r.URL.Query().Get("page"); v != "" {
				if n, err := strconv.Atoi(v); err == nil && n > 0 {
					page = n
				}
			}
			if v := r.URL.Query().Get("page_size"); v != "" {
				if n, err := strconv.Atoi(v); err == nil && n > 0 {
					pageSize = n
				}
			}

			resp, err := client.ListPosts(context.Background(), &proto.ListPostsRequest{
				UserId:   userID,
				Page:     int32(page),
				PageSize: int32(pageSize),
			})
			if err != nil {
				http.Error(w, "service error", http.StatusBadGateway)
				return
			}

			respondJSON(w, http.StatusOK, resp)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func PostsWithID(client proto.PostsServiceClient) http.HandlerFunc {
	return AuthMiddleware(func(w http.ResponseWriter, r *http.Request, userID string) {
		id := strings.TrimPrefix(r.URL.Path, "/posts/")
		if id == "" || strings.Contains(id, "/") {
			http.NotFound(w, r)
			return
		}

		switch r.Method {
		case http.MethodGet:
			resp, err := client.GetPost(context.Background(), &proto.GetPostRequest{
				Id:     id,
				UserId: userID,
			})
			if err != nil {
				http.Error(w, "service error", http.StatusBadGateway)
				return
			}

			respondJSON(w, http.StatusOK, resp.Post)
		case http.MethodPut:
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
		case http.MethodDelete:
			resp, err := client.DeletePost(context.Background(), &proto.DeletePostRequest{
				Id:     id,
				UserId: userID,
			})
			if err != nil {
				http.Error(w, "service error", http.StatusBadGateway)
				return
			}

			respondJSON(w, http.StatusOK, resp)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}
