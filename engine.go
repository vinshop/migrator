package migrator

import (
	"context"
	"fmt"
	"github.com/fatih/color"
	"github.com/vinshop/migration/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"sort"
)

type Engine struct {
	ctx            context.Context
	db             *mongo.Database
	migrations     []Migrator
	metaCollection *mongo.Collection
}

func New(ctx context.Context, db *mongo.Database) *Engine {
	return &Engine{
		ctx:            ctx,
		db:             db,
		migrations:     nil,
		metaCollection: nil,
	}
}

func (e *Engine) Register(version string, withTransaction bool, fn MigrationFunc) {
	e.migrations = append(e.migrations, Migrator{withTransaction, version, fn})
}

func (e *Engine) updateCurrentVersion(version string, dirty bool) error {
	if _, err := e.metaCollection.DeleteMany(e.ctx, bson.M{}); err != nil {
		return err
	}
	if _, err := e.metaCollection.InsertOne(e.ctx, model.MigrationMeta{CurrentVersion: version, Dirty: dirty}); err != nil {
		return err
	}
	return nil
}

func (e *Engine) fetchCurrentVersion() (*model.MigrationMeta, error) {
	meta := &model.MigrationMeta{
		ID:             primitive.ObjectID{},
		CurrentVersion: "0",
		Dirty:          false,
	}
	if err := e.db.Collection(model.MigrationMeta{}.CollectionName()).FindOne(e.ctx, bson.M{}).Decode(&meta); err != nil {
		if err == mongo.ErrNoDocuments {
			color.Yellow("no version found, running from beginning")
			return meta, nil
		}
		return nil, err
	}
	return meta, nil
}

func (e *Engine) migrate(m Migrator) error {
	ctx := context.TODO()
	if !m.Transaction {
		return m.Func(e.ctx, e.db, m.Version)
	}
	session, err := e.db.Client().StartSession()
	if err != nil {
		return err
	}
	if err := session.StartTransaction(); err != nil {
		return err
	}
	defer session.EndSession(ctx)
	return mongo.WithSession(ctx, session, func(sessionContext mongo.SessionContext) error {
		if err := m.Func(sessionContext, e.db, m.Version); err != nil {
			return err
		}
		return session.CommitTransaction(sessionContext)
	})
}

func (e *Engine) doMigration(currentVer *model.MigrationMeta, version Migrator) error {
	s := color.BlueString("version: %v => ", version.Version)
	defer func() {
		fmt.Println(s)
	}()
	if version.Version > currentVer.CurrentVersion {
		currentVer.CurrentVersion = version.Version
		if err := e.migrate(version); err != nil {
			s += color.RedString(err.Error())
			if err := e.updateCurrentVersion(currentVer.CurrentVersion, true); err != nil {
				color.Red("error when update current version to %v, dirty: %v, error: %v", currentVer.CurrentVersion, true, err)
				return err
			}
			return err
		}
		if err := e.updateCurrentVersion(currentVer.CurrentVersion, false); err != nil {
			color.Red("error when update current version to %v, dirty: %v, error: %v", currentVer.CurrentVersion, false, err)
			return err
		}
		s += color.GreenString("done")
	} else {
		s += color.YellowString("skip")
	}
	return nil
}

func (e *Engine) Run() error {
	e.metaCollection = e.db.Collection(model.MigrationMeta{}.CollectionName())
	color.Green("sorting version")
	sort.Slice(e.migrations, func(i, j int) bool {
		return e.migrations[i].Version < e.migrations[j].Version
	})
	currentVer, err := e.fetchCurrentVersion()
	if err != nil {
		color.Red("error when get current version, error:", err.Error())
	}
	if currentVer.Dirty {
		color.Red("dirty version found, please resolve dirty and roll back to previous version")
		return nil
	}
	color.Blue("current version: %v", currentVer.CurrentVersion)
	for _, version := range e.migrations {
		if err := e.doMigration(currentVer, version); err != nil {
			return err
		}
	}

	color.Green("done")
	return nil
}
