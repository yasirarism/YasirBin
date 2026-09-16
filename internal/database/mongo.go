package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"yasirbin/internal/model"
)

type MongoStore struct {
	client *mongo.Client
	db     *mongo.Database
	coll   *mongo.Collection
}

type mongoDoc struct {
	Slug      string     `bson:"slug"`
	Content   string     `bson:"content"`
	Password  string     `bson:"password"`
	CreatedAt time.Time  `bson:"created_at"`
	ExpiresAt *time.Time `bson:"expires_at,omitempty"`
}

func NewMongo(uri, dbName string) (*MongoStore, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("mongo ping: %w", err)
	}

	db := client.Database(dbName)
	coll := db.Collection("documents")

	store := &MongoStore{
		client: client,
		db:     db,
		coll:   coll,
	}

	if err := store.ensureIndexes(); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("mongo indexes: %w", err)
	}

	return store, nil
}

func (m *MongoStore) Driver() string {
	return "mongodb"
}

func (m *MongoStore) ensureIndexes() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	indexModels := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "slug", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "expires_at", Value: 1}},
		},
	}

	_, err := m.coll.Indexes().CreateMany(ctx, indexModels)
	return err
}

func (m *MongoStore) Create(doc *model.Document) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	data := mongoDoc{
		Slug:      doc.Slug,
		Content:   doc.Content,
		Password:  doc.Password,
		CreatedAt: doc.CreatedAt,
		ExpiresAt: doc.ExpiresAt,
	}

	_, err := m.coll.InsertOne(ctx, data)
	return err
}

func (m *MongoStore) GetBySlug(slug string) (*model.Document, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var data mongoDoc
	err := m.coll.FindOne(ctx, bson.M{"slug": slug}).Decode(&data)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &model.Document{
		Slug:      data.Slug,
		Content:   data.Content,
		Password:  data.Password,
		CreatedAt: data.CreatedAt,
		ExpiresAt: data.ExpiresAt,
	}, nil
}

func (m *MongoStore) SlugExists(slug string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := m.coll.CountDocuments(ctx, bson.M{"slug": slug})
	return err == nil && count > 0
}

func (m *MongoStore) CleanExpired() int64 {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	now := time.Now().UTC()
	filter := bson.M{
		"expires_at": bson.M{
			"$ne": nil,
			"$lt": now,
		},
	}

	res, err := m.coll.DeleteMany(ctx, filter)
	if err != nil {
		log.Printf("[mongodb] cleanup error: %v", err)
		return 0
	}
	return res.DeletedCount
}

func (m *MongoStore) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return m.client.Disconnect(ctx)
}
