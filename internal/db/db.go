package db

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	// ConnectionTimeout is the default timeout for connecting to the database.
	ConnectionTimeout = 60 * time.Second

	// DatabaseName is the name of the database
	DatabaseName = "ado-asana-sync"

	// Timeout is the default timeout for database operations.
	Timeout = 10 * time.Second
)

type DB struct {
	Client *mongo.Client
}

func (db *DB) Connect(uri string) error {
	ctx, cancel := context.WithTimeout(context.Background(), ConnectionTimeout)
	defer cancel()

	c, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}
	db.Client = c

	err = db.Client.Ping(ctx, nil)
	if err != nil {
		return err
	}

	log.Println("Connected to MongoDB!")
	return nil
}
