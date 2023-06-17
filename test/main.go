package main

import (
	"context"
	"fmt"
	migrator "github.com/vinshop/migrator"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Source struct {
	URI          string `mapstructure:"MONGO_DB_URI"`
	DatabaseName string `mapstructure:"MONGO_DB_NAME"`
}

func NewMongoConnection(config *Source) (*mongo.Database, func() error, error) {
	client, err := mongo.NewClient(options.Client().ApplyURI(config.URI))
	if err != nil {
		return nil, nil, err
	}
	ctx := context.Background()
	err = client.Connect(ctx)
	if err != nil {
		return nil, nil, err
	}
	if err := client.Ping(context.TODO(), readpref.Primary()); err != nil {
		return nil, nil, err
	}
	return client.Database(config.DatabaseName), func() error {
		return client.Disconnect(ctx)
	}, nil
}

func main() {
	config := Source{
		URI:          "mongodb://root:root@127.0.0.1:27018/test_migrator?authSource=admin",
		DatabaseName: "test_migrator",
	}

	db, closer, err := NewMongoConnection(&config)
	if err != nil {
		panic(err)
	}
	defer closer()
	engine := migrator.New(context.TODO(), db)

	engine.Register("1", true, func(ctx context.Context, db *mongo.Database, version string) error {
		return nil
	})

	engine.Register("2", false, func(ctx context.Context, db *mongo.Database, version string) error {
		return nil
	})

	engine.Register("3", true, func(ctx context.Context, db *mongo.Database, version string) error {
		return nil
	})

	engine.Register("4", true, func(ctx context.Context, db *mongo.Database, version string) error {
		if err := migrator.Batch(100, 10, func(skip int, limit int) error {
			fmt.Printf("skip %v, limit %v\n", skip, limit)
			return nil
		}); err != nil {
			return err
		}
		return migrator.DoNothing
	})

	engine.Run()

}
