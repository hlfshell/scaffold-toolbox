package mongo

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/hlfshell/scaffold"
	scaffoldcontainer "github.com/hlfshell/scaffold/container"
	"github.com/hlfshell/scaffold/logs"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

/*
Mongo is a typed harness around the official MongoDB container. It keeps
connection settings, an optional client, and document preload functions.
*/
type Mongo struct {
	container *scaffoldcontainer.Container
	client    *mongodriver.Client
	name      string
	username  string
	password  string
	database  string
	port      string
	preloads  []func(context.Context, *Mongo) error
}

/*
NewMongo creates a MongoDB service. If username and password are set,
the container is initialized with root credentials.
*/
func NewMongo(name string, tag string, username string, password string, database string) (*Mongo, error) {
	env := map[string]string{}
	if username != "" || password != "" {
		env["MONGO_INITDB_ROOT_USERNAME"] = username
		env["MONGO_INITDB_ROOT_PASSWORD"] = password
	}
	if database != "" {
		env["MONGO_INITDB_DATABASE"] = database
	}

	container, err := scaffoldcontainer.NewContainer(
		name,
		"mongo",
		scaffoldcontainer.WithTag(tag),
		scaffoldcontainer.WithPort("27017", ""),
		scaffoldcontainer.WithEnv(env),
	)
	if err != nil {
		return nil, err
	}

	return &Mongo{
		container: container,
		name:      name,
		username:  username,
		password:  password,
		database:  database,
		preloads:  []func(context.Context, *Mongo) error{},
	}, nil
}

func (m *Mongo) Name() string {
	return m.name
}

/*
SetNetwork attaches the underlying MongoDB container to a shared Docker
network.
*/
func (m *Mongo) SetNetwork(name string) {
	m.container.SetNetwork(name)
}

/*
SetLabels merges inherited Docker labels onto the MongoDB container.
*/
func (m *Mongo) SetLabels(labels map[string]string) {
	m.container.SetLabels(labels)
}

/*
SetNamePrefix prefixes the Docker container name before it is created.
*/
func (m *Mongo) SetNamePrefix(prefix string) {
	m.container.SetNamePrefix(prefix)
}

/*
Create starts MongoDB, waits until it accepts pings, and runs registered
document preload functions.
*/
func (m *Mongo) Create(ctx context.Context) error {
	err := m.container.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start mongo container: %w", err)
	}

	ports := m.container.GetPorts()
	m.port = ports["27017"]

	_, err = m.connectWithTimeoutContext(ctx, 45*time.Second)
	if err != nil {
		m.container.Cleanup(context.WithoutCancel(ctx))
		return fmt.Errorf("mongo failed to become ready: %w", err)
	}

	err = m.Preload(ctx)
	if err != nil {
		m.container.Cleanup(context.WithoutCancel(ctx))
		return err
	}

	return nil
}

/*
Connect opens a MongoDB client with ctx and verifies it with Ping.
*/
func (m *Mongo) Connect(ctx context.Context) (*mongodriver.Client, error) {
	return m.connectContext(ctx)
}

/*
ConnectWithTimeout retries Connect until MongoDB is ready or timeout is
reached.
*/
func (m *Mongo) ConnectWithTimeout(ctx context.Context, timeout time.Duration) (*mongodriver.Client, error) {
	return m.connectWithTimeoutContext(ctx, timeout)
}

func (m *Mongo) connectWithTimeoutContext(ctx context.Context, timeout time.Duration) (*mongodriver.Client, error) {
	var client *mongodriver.Client

	err := scaffold.WaitFunc(ctx, timeout, 100*time.Millisecond, func(ctx context.Context) error {
		var err error
		client, err = m.connectContext(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (m *Mongo) connectContext(ctx context.Context) (*mongodriver.Client, error) {
	client, err := mongodriver.Connect(ctx, options.Client().ApplyURI(m.ConnectionString()))
	if err != nil {
		return nil, err
	}

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		_ = client.Disconnect(ctx)
		return nil, err
	}

	if m.client != nil {
		_ = m.client.Disconnect(context.Background())
	}
	m.client = client

	return client, nil
}

/*
ConnectionString returns a localhost MongoDB URI for the assigned host
port.
*/
func (m *Mongo) ConnectionString() string {
	host := fmt.Sprintf("127.0.0.1:%s", m.port)
	if m.username == "" && m.password == "" {
		if m.database == "" {
			return "mongodb://" + host
		}
		return fmt.Sprintf("mongodb://%s/%s", host, url.PathEscape(m.database))
	}

	user := url.QueryEscape(m.username)
	pass := url.QueryEscape(m.password)
	db := m.database
	if db == "" {
		db = "admin"
	}

	return fmt.Sprintf("mongodb://%s:%s@%s/%s?authSource=admin", user, pass, host, url.PathEscape(db))
}

func (m *Mongo) Env() map[string]string {
	return map[string]string{
		"MONGO_URL": m.ConnectionString(),
	}
}

func (m *Mongo) Endpoints() map[string]string {
	return map[string]string{
		m.name: fmt.Sprintf("127.0.0.1:%s", m.port),
	}
}

/*
Client returns the last successful MongoDB client.
*/
func (m *Mongo) Client() *mongodriver.Client {
	return m.client
}

/*
InsertDocuments inserts documents into the configured database.
*/
func (m *Mongo) InsertDocuments(ctx context.Context, collection string, documents ...any) error {
	client, err := m.connectWithTimeoutContext(ctx, 10*time.Second)
	if err != nil {
		return err
	}

	database := m.database
	if database == "" {
		database = "test"
	}

	_, err = client.Database(database).Collection(collection).InsertMany(ctx, documents)
	return err
}

/*
WithDocuments registers documents to insert after MongoDB is ready.
*/
func (m *Mongo) WithDocuments(collection string, documents ...any) *Mongo {
	m.preloads = append(m.preloads, func(ctx context.Context, mongo *Mongo) error {
		return mongo.InsertDocuments(ctx, collection, documents...)
	})

	return m
}

/*
WithSeed registers a custom MongoDB seed function.
*/
func (m *Mongo) WithSeed(fn func(context.Context, *Mongo) error) *Mongo {
	m.preloads = append(m.preloads, fn)
	return m
}

/*
Preload runs all registered MongoDB seed functions.
*/
func (m *Mongo) Preload(ctx context.Context) error {
	for _, preload := range m.preloads {
		err := preload(ctx, m)
		if err != nil {
			return fmt.Errorf("failed to preload mongo: %w", err)
		}
	}

	return nil
}

/*
Cleanup disconnects the client and removes the MongoDB container.
*/
func (m *Mongo) Cleanup(ctx context.Context) error {
	if m.client != nil {
		_ = m.client.Disconnect(ctx)
		m.client = nil
	}

	return m.container.Cleanup(ctx)
}

/*
Logs returns MongoDB container logs keyed by service name.
*/
func (m *Mongo) Logs(ctx context.Context) (logs.LogStreams, error) {
	stream, err := m.container.Logs(ctx)
	if err != nil {
		return nil, err
	}

	return logs.LogStreams{m.name: stream}, nil
}
