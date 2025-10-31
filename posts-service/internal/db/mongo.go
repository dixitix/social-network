package db

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("not found")

type Post struct {
	ID        string    `bson:"id"`
	OwnerID   string    `bson:"owner_id"`
	Title     string    `bson:"title"`
	Content   string    `bson:"content"`
	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
}

type DB struct {
	coll *mongo.Collection
}

func New(client *mongo.Client, dbName, collName string) *DB {
	return &DB{coll: client.Database(dbName).Collection(collName)}
}

func (db *DB) Create(p Post) (Post, error) {
	ctx := context.Background()

	p.CreatedAt = time.Now().UTC()
	p.UpdatedAt = p.CreatedAt
	_, err := db.coll.InsertOne(ctx, p)
	return p, err
}

func (db *DB) Update(id, ownerID, title, content string) (Post, error) {
	ctx := context.Background()

	filter := bson.D{{Key: "id", Value: id}, {Key: "owner_id", Value: ownerID}}
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "title", Value: title},
		{Key: "content", Value: content},
		{Key: "updated_at", Value: time.Now().UTC()},
	}}}

	var out Post
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	err := db.coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&out)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return Post{}, ErrNotFound
	}
	return out, err
}

func (db *DB) Delete(id, ownerID string) error {
	ctx := context.Background()

	_, err := db.coll.DeleteOne(ctx, bson.D{{Key: "id", Value: id}, {Key: "owner_id", Value: ownerID}})
	return err
}

func (db *DB) Get(id, ownerID string) (Post, error) {
	ctx := context.Background()

	var p Post
	err := db.coll.FindOne(ctx, bson.D{{Key: "id", Value: id}, {Key: "owner_id", Value: ownerID}}).Decode(&p)
	return p, err
}

func (db *DB) List(ownerID string, limit int64) ([]Post, error) {
	ctx := context.Background()

	findOpts := options.Find()
	if limit > 0 {
		findOpts.SetLimit(limit)
	}

	cur, err := db.coll.Find(ctx, bson.D{{Key: "owner_id", Value: ownerID}}, findOpts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var posts []Post
	for cur.Next(ctx) {
		var p Post
		if err := cur.Decode(&p); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, cur.Err()
}

func NewStringID() string {
	return uuid.New().String()
}
