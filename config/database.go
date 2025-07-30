package config

import (
	"context"
	"fmt"
	// "log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ConnectDB menginisialisasi koneksi ke MongoDB.
func ConnectDB(uri string, mode string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("error connecting to MongoDB: %w", err)
	}

	if err = client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("error pinging MongoDB: %w", err)
	}

	if mode == "atlas" {
		fmt.Println("🌐 Successfully connected to MongoDB Atlas")
	} else {
		fmt.Println("🏠 Successfully connected to Local MongoDB")
	}

	return client, nil
}