package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"

	proto "posts-service/proto"
	statspb "stats-service/proto"
)

func StatsPost(client statspb.StatsServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}

		resp, err := client.GetPostStats(r.Context(), &statspb.PostStatsRequest{PostId: id})
		if err != nil {
			http.Error(w, "service error", http.StatusBadGateway)
			return
		}

		if resp == nil {
			resp = &statspb.PostStatsResponse{PostId: id}
		}

		respondJSON(w, http.StatusOK, map[string]any{
			"id":    resp.GetPostId(),
			"views": resp.GetViews(),
			"likes": resp.GetLikes(),
		})
	}
}

func StatsTopPosts(statsClient statspb.StatsServiceClient, postsClient proto.PostsServiceClient, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		metric := r.URL.Query().Get("metric")
		if metric == "" {
			metric = "views"
		}
		if metric != "views" && metric != "likes" {
			http.Error(w, "invalid metric", http.StatusBadRequest)
			return
		}

		resp, err := statsClient.GetTopPosts(r.Context(), &statspb.TopPostsRequest{Metric: metric, Limit: 5})
		if err != nil {
			http.Error(w, "service error", http.StatusBadGateway)
			return
		}

		type item struct {
			ID          string `json:"id"`
			AuthorLogin string `json:"author_login"`
			Value       int64  `json:"value"`
		}
		var out []item
		for _, post := range resp.GetItems() {
			postResp, err := postsClient.GetPost(r.Context(), &proto.GetPostRequest{Id: post.GetPostId()})
			if err != nil || postResp == nil || postResp.Post == nil {
				continue
			}
			login, err := lookupLogin(r.Context(), db, postResp.Post.GetOwnerId())
			if err != nil {
				continue
			}
			out = append(out, item{
				ID:          post.GetPostId(),
				AuthorLogin: login,
				Value:       post.GetValue(),
			})
		}

		respondJSON(w, http.StatusOK, out)
	}
}

func StatsTopUsers(statsClient statspb.StatsServiceClient, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		resp, err := statsClient.GetTopUsersByLikes(r.Context(), &statspb.TopUsersRequest{Limit: 3})
		if err != nil {
			http.Error(w, "service error", http.StatusBadGateway)
			return
		}

		type item struct {
			Login string `json:"login"`
			Likes int64  `json:"likes"`
		}
		var out []item
		for _, user := range resp.GetUsers() {
			login, err := lookupLogin(r.Context(), db, user.GetUserId())
			if err != nil {
				continue
			}
			out = append(out, item{Login: login, Likes: user.GetLikes()})
		}

		respondJSON(w, http.StatusOK, out)
	}
}

func lookupLogin(ctx context.Context, db *sql.DB, userID string) (string, error) {
	if userID == "" {
		return "", sql.ErrNoRows
	}
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return "", err
	}
	var login string
	err = db.QueryRowContext(ctx, `select login from users where id = $1`, id).Scan(&login)
	if err != nil {
		return "", err
	}
	return login, nil
}
