package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type Config struct {
	Addr     []string
	DB       string
	User     string
	Password string
}

type Event struct {
	PostID    string
	EventType string
	Timestamp time.Time
}

type Repository struct {
	conn   driver.Conn
	dbName string
}

func New(ctx context.Context, cfg Config) (*Repository, error) {
	if len(cfg.Addr) == 0 {
		cfg.Addr = []string{"stats-clickhouse:9000"}
	}
	if cfg.DB == "" {
		cfg.DB = "stats"
	}
	opts := &ch.Options{
		Addr: cfg.Addr,
		Auth: ch.Auth{
			Database: cfg.DB,
			Username: cfg.User,
			Password: cfg.Password,
		},
		DialTimeout:     5 * time.Second,
		ConnMaxLifetime: 10 * time.Minute,
	}

	var (
		conn driver.Conn
		err  error
	)
	for attempt := 0; attempt < 5; attempt++ {
		conn, err = ch.Open(opts)
		if err == nil {
			pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err = conn.Ping(pingCtx)
			cancel()
			if err == nil {
				break
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				// respect caller cancellation immediately
				conn.Close()
				return nil, fmt.Errorf("clickhouse ping cancelled: %w", err)
			}
			conn.Close()
		}

		wait := time.Duration(attempt+1) * time.Second
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("clickhouse connect aborted: %w", ctx.Err())
		case <-time.After(wait):
		}
	}
	if err != nil {
		return nil, fmt.Errorf("clickhouse connect failed: %w", err)
	}

	repo := &Repository{
		conn:   conn,
		dbName: cfg.DB,
	}
	if err := repo.ensureSchema(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ensure schema failed: %w", err)
	}
	return repo, nil
}

func (r *Repository) Close() error {
	if r == nil || r.conn == nil {
		return nil
	}
	return r.conn.Close()
}

func (r *Repository) ensureSchema(ctx context.Context) error {
	if err := r.conn.Exec(ctx, "CREATE DATABASE IF NOT EXISTS "+r.dbName); err != nil {
		return err
	}
	createTable := "CREATE TABLE IF NOT EXISTS " + r.dbName + `.events (
    event_type String,
    post_id String,
    ts DateTime
) ENGINE = MergeTree() ORDER BY (event_type, post_id, ts)
`
	return r.conn.Exec(ctx, createTable)
}

func (r *Repository) SaveEvent(ctx context.Context, e Event) error {
	query := "INSERT INTO " + r.dbName + ".events (event_type, post_id, ts) VALUES (?, ?, ?)"
	return r.conn.Exec(ctx, query, e.EventType, e.PostID, e.Timestamp)
}
