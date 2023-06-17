package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type MigrationMeta struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	CurrentVersion string             `bson:"current_version"`
	Dirty          bool               `bson:"dirty"`
}

func (MigrationMeta) CollectionName() string {
	return "_migration"
}
