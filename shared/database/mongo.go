package database

import (
	"context"
	"fmt"
	"time"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/dbid"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type MongoDB struct {
	Client *mongo.Client
	DB     *mongo.Database
}

// NewMongoDB connects with the correct BSON codec for the configured ID strategy
// (uuid or objectid) and retries until the replica-set primary is available.
func NewMongoDB(cfg config.MongoConfig) (*MongoDB, error) {
	return NewMongoDBWithMonitor(cfg, nil)
}

// NewMongoDBWithMonitor is NewMongoDB plus an optional *event.CommandMonitor
// (see pkg/metrics.MongoCommandMonitor) for per-command logging/metrics. A
// nil monitor behaves exactly like NewMongoDB — only long-running processes
// that want Prometheus/zap visibility into MongoDB traffic (apps/api) need
// this; one-shot scripts (scripts/seed, scripts/migrate) can keep calling
// NewMongoDB.
func NewMongoDBWithMonitor(cfg config.MongoConfig, monitor *event.CommandMonitor) (*MongoDB, error) {
	clientOpts := []*options.ClientOptions{
		options.Client().ApplyURI(cfg.URI),
		// Register the correct _id codec based on DB_ID_TYPE config
		options.Client().SetRegistry(dbid.NewRegistry(cfg.IDType)),
		options.Client().
			SetMinPoolSize(cfg.PoolMin).
			SetMaxPoolSize(cfg.PoolMax).
			SetConnectTimeout(cfg.Timeout).
			SetServerSelectionTimeout(cfg.Timeout),
	}
	if monitor != nil {
		clientOpts = append(clientOpts, options.Client().SetMonitor(monitor))
	}
	opts := options.MergeClientOptions(clientOpts...)

	client, err := mongo.Connect(context.Background(), opts)
	if err != nil {
		return nil, fmt.Errorf("mongo: connect: %w", err)
	}

	const maxAttempts = 10
	const retryDelay = 3 * time.Second
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = client.Ping(ctx, readpref.Primary())
		cancel()
		if err == nil {
			break
		}
		if attempt == maxAttempts {
			_ = client.Disconnect(context.Background())
			return nil, fmt.Errorf("mongo: ping failed after %d attempts: %w", maxAttempts, err)
		}
		fmt.Printf("mongo: waiting for primary (attempt %d/%d)...\n", attempt, maxAttempts)
		time.Sleep(retryDelay)
	}

	return &MongoDB{Client: client, DB: client.Database(cfg.DBName)}, nil
}

func (m *MongoDB) Disconnect(ctx context.Context) error         { return m.Client.Disconnect(ctx) }
func (m *MongoDB) Collection(name string) *mongo.Collection     { return m.DB.Collection(name) }
func (m *MongoDB) Ping(ctx context.Context) error               { return m.Client.Ping(ctx, readpref.Primary()) }

func (m *MongoDB) EnsureIndexes(ctx context.Context, col string, models []mongo.IndexModel) error {
	_, err := m.DB.Collection(col).Indexes().CreateMany(ctx, models)
	return err
}

type SortDirection int
const (
	ASC  SortDirection = 1
	DESC SortDirection = -1
)

func BuildSortDoc(fields []string, dir SortDirection) bson.D {
	d := make(bson.D, 0, len(fields))
	for _, f := range fields {
		d = append(d, bson.E{Key: f, Value: int(dir)})
	}
	return d
}

type Transactor struct{ client *mongo.Client }

func NewTransactor(m *MongoDB) *Transactor { return &Transactor{client: m.Client} }

func (t *Transactor) WithTransaction(ctx context.Context, fn func(sc mongo.SessionContext) (interface{}, error)) error {
	session, err := t.client.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, fn)
	return err
}
