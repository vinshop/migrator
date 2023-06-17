package migrator

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
)

type MigrationFunc func(ctx context.Context, db *mongo.Database, version string) error

type Migrator struct {
	Transaction bool
	Version     string
	Func        MigrationFunc
}
