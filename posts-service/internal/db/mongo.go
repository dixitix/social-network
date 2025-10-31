package db

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

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
	p.CreatedAt = time.Now().UTC()
	p.UpdatedAt = p.CreatedAt
	_, err := db.coll.InsertOne(nil, p)
	return p, err
}

func (db *DB) Update(id, ownerID, title, content string) (Post, error) {
	filter := bson.D{{Key: "id", Value: id}, {Key: "owner_id", Value: ownerID}}
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "title", Value: title},
		{Key: "content", Value: content},
		{Key: "updated_at", Value: time.Now().UTC()},
	}}}

	var out Post
	err := db.coll.FindOneAndUpdate(nil, filter, update).Decode(&out)
	return out, err
}

func (db *DB) Delete(id, ownerID string) error {
	_, err := db.coll.DeleteOne(nil, bson.D{{Key: "id", Value: id}, {Key: "owner_id", Value: ownerID}})
	return err
}

func (db *DB) Get(id, ownerID string) (Post, error) {
	var p Post
	err := db.coll.FindOne(nil, bson.D{{Key: "id", Value: id}, {Key: "owner_id", Value: ownerID}}).Decode(&p)
	return p, err
}

func (db *DB) List(ownerID string, limit int64) ([]Post, error) {
	cur, err := db.coll.Find(nil, bson.D{{Key: "owner_id", Value: ownerID}}, nil)
	if err != nil {
		return nil, err
	}
	defer cur.Close(nil)

	var posts []Post
	for cur.Next(nil) {
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
