package database

import (
	"context"
	"fmt"
	"strings"
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
	// Deployment is the RESOLVED topology — "standalone", "replicaset" or
	// "sharded" — never the literal "auto" the caller may have configured.
	Deployment string
	// Transactions reports whether multi-document transactions are available on
	// this deployment. Resolved ONCE here, at connect time, and read from
	// NewTransactor — the single-resolver shape, so a caller cannot construct a
	// Transactor that disagrees with the server it is talking to.
	Transactions bool
}

// Deployment modes accepted in MONGO_DEPLOYMENT.
const (
	DeploymentAuto       = "auto"
	DeploymentStandalone = "standalone"
	DeploymentReplicaSet = "replicaset"
	DeploymentSharded    = "sharded"
)

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

	deployment, transactions := resolveDeployment(client, cfg.Deployment)

	return &MongoDB{
		Client:       client,
		DB:           client.Database(cfg.DBName),
		Deployment:   deployment,
		Transactions: transactions,
	}, nil
}

// resolveDeployment decides whether multi-document transactions are available.
//
// A STANDALONE mongod refuses them outright ("Transaction numbers are only
// allowed on a replica set member or mongos"), so without this the failure
// surfaces deep inside the first transactional write on a deployment that
// answers every read and every other write perfectly. Configuring the mode
// explicitly is supported for when the probe cannot be trusted; "auto" is the
// default and keeps a replica set from being silently downgraded by a stale env
// file.
func resolveDeployment(client *mongo.Client, configured string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(configured)) {
	case DeploymentStandalone:
		return DeploymentStandalone, false
	case DeploymentReplicaSet:
		return DeploymentReplicaSet, true
	case DeploymentSharded:
		return DeploymentSharded, true
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// `hello` replaced `isMaster` in 5.0; fall back for anything older rather
	// than treating a CommandNotFound as "standalone", which would disable
	// transactions on a perfectly good replica set.
	var res bson.M
	err := client.Database("admin").RunCommand(ctx, bson.D{{Key: "hello", Value: 1}}).Decode(&res)
	if err != nil {
		res = nil
		if e2 := client.Database("admin").RunCommand(ctx, bson.D{{Key: "isMaster", Value: 1}}).Decode(&res); e2 != nil {
			// Probe failed entirely. Assume transactions ARE available: this is
			// the safe direction — a real transaction against a standalone fails
			// loudly and immediately, whereas assuming standalone would quietly
			// drop atomicity on a replica set and nothing would look wrong until
			// two writes raced.
			return DeploymentReplicaSet, true
		}
	}
	if msg, _ := res["msg"].(string); msg == "isdbgrid" {
		return DeploymentSharded, true // mongos — transactions supported
	}
	if name, _ := res["setName"].(string); name != "" {
		return DeploymentReplicaSet, true
	}
	return DeploymentStandalone, false
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

type Transactor struct {
	client       *mongo.Client
	transactions bool
}

func NewTransactor(m *MongoDB) *Transactor {
	return &Transactor{client: m.Client, transactions: m.Transactions}
}

// NewTransactorWithMode builds a Transactor against an explicit mode. Tests use
// it; application code takes the resolved value off *MongoDB via NewTransactor.
func NewTransactorWithMode(client *mongo.Client, transactions bool) *Transactor {
	return &Transactor{client: client, transactions: transactions}
}

// TransactionsEnabled reports whether WithTransaction is actually atomic. A
// caller that needs the guarantee — rather than merely the grouping of
// reads/writes into one session — must check this and defend itself.
func (t *Transactor) TransactionsEnabled() bool { return t.transactions }

func (t *Transactor) WithTransaction(ctx context.Context, fn func(sc mongo.SessionContext) (interface{}, error)) error {
	session, err := t.client.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	// Standalone deployments have no transactions to start. The closure still
	// runs inside the SESSION — so reads and writes stay causally consistent —
	// but it is NOT atomic and nothing rolls back on error. Callers that depend
	// on atomicity must guard themselves; see TransactionsEnabled.
	if !t.transactions {
		_, err = fn(mongo.NewSessionContext(ctx, session))
		return err
	}

	_, err = session.WithTransaction(ctx, fn)
	return err
}
