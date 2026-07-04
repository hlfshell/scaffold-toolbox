package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/hlfshell/scaffold"
	scaffoldcontainer "github.com/hlfshell/scaffold/container"
	"github.com/hlfshell/scaffold/logs"

	_ "github.com/lib/pq"
)

/*
Postgres is a typed harness around the official postgres container. It
keeps the assigned port, connection settings, and preload functions.
*/
type Postgres struct {
	container *scaffoldcontainer.Container
	db        *sql.DB
	name      string
	username  string
	password  string
	database  string
	port      string
	preloads  []func(*sql.DB) error
}

/*
NewPostgres creates a Postgres harness. A blank tag is passed through to
Scaffold and will default to "latest".
*/
func NewPostgres(name string, tag string, username string, password string, database string) (*Postgres, error) {
	container, err := scaffoldcontainer.NewContainer(
		name,
		"postgres",
		scaffoldcontainer.WithTag(tag),
		scaffoldcontainer.WithPort("5432", ""),
		scaffoldcontainer.WithEnv(map[string]string{
			"POSTGRES_USER":     username,
			"POSTGRES_PASSWORD": password,
			"POSTGRES_DB":       database,
		}),
	)
	if err != nil {
		return nil, err
	}

	return &Postgres{
		container: container,
		name:      name,
		username:  username,
		password:  password,
		database:  database,
		preloads:  []func(*sql.DB) error{},
	}, nil
}

/*
Name returns the service name used by Scaffold stacks.
*/
func (p *Postgres) Name() string {
	return p.name
}

/*
SetNetwork attaches the underlying container to a Docker network when it
is created.
*/
func (p *Postgres) SetNetwork(name string) {
	p.container.SetNetwork(name)
}

/*
SetLabels merges Docker labels onto the underlying container.
*/
func (p *Postgres) SetLabels(labels map[string]string) {
	p.container.SetLabels(labels)
}

/*
SetNamePrefix prefixes the underlying Docker container name before it is
created.
*/
func (p *Postgres) SetNamePrefix(prefix string) {
	p.container.SetNamePrefix(prefix)
}

/*
Create starts Postgres with ctx, waits until it accepts connections, and runs
any registered preload functions.
*/
func (p *Postgres) Create(ctx context.Context) error {
	err := p.container.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start postgres container: %w", err)
	}

	ports := p.container.GetPorts()
	p.port = ports["5432"]

	err = scaffold.WaitFunc(ctx, 30*time.Second, 50*time.Millisecond, func(ctx context.Context) error {
		db, err := p.connectContext(ctx)
		if err != nil {
			return err
		}

		return db.PingContext(ctx)
	})
	if err != nil {
		p.container.Cleanup(context.WithoutCancel(ctx))
		return fmt.Errorf("postgres failed to become ready: %w", err)
	}

	err = p.Preload()
	if err != nil {
		p.container.Cleanup(context.WithoutCancel(ctx))
		return err
	}

	return nil
}

/*
ConnectWithTimeout repeatedly calls Connect until a connection succeeds
or the timeout is reached.
*/
func (p *Postgres) ConnectWithTimeout(timeout time.Duration) (*sql.DB, error) {
	var db *sql.DB

	err := scaffold.WaitFunc(context.Background(), timeout, 50*time.Millisecond, func(ctx context.Context) error {
		var err error
		db, err = p.connectContext(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}

	return db, nil
}

/*
Connect opens and pings a database connection using the assigned host
port.
*/
func (p *Postgres) Connect() (*sql.DB, error) {
	return p.connectContext(context.Background())
}

func (p *Postgres) connectContext(ctx context.Context) (*sql.DB, error) {
	connectionString := p.ConnectionString()

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, err
	}

	err = db.PingContext(ctx)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	if p.db != nil {
		p.db.Close()
	}
	p.db = db

	return db, nil
}

/*
ConnectionString returns the local Postgres connection string.
*/
func (p *Postgres) ConnectionString() string {
	return fmt.Sprintf(
		"postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		p.username,
		p.password,
		"127.0.0.1",
		p.port,
		p.database,
	)
}

/*
Port returns the assigned host port for Postgres.
*/
func (p *Postgres) Port() string {
	return p.port
}

/*
Env returns Postgres connection environment variables.
*/
func (p *Postgres) Env() map[string]string {
	return map[string]string{
		"POSTGRES_URL": p.ConnectionString(),
	}
}

/*
Endpoints returns named Postgres endpoints.
*/
func (p *Postgres) Endpoints() map[string]string {
	return map[string]string{
		p.name: "127.0.0.1:" + p.port,
	}
}

/*
GetDB returns the last successful database connection.
*/
func (p *Postgres) GetDB() *sql.DB {
	return p.db
}

/*
GetContainer returns the underlying Scaffold container.
*/
func (p *Postgres) GetContainer() *scaffoldcontainer.Container {
	return p.container
}

/*
WithSQL registers a SQL string to run after Postgres is ready.
*/
func (p *Postgres) WithSQL(query string) *Postgres {
	p.preloads = append(p.preloads, func(db *sql.DB) error {
		_, err := db.Exec(query)
		return err
	})

	return p
}

/*
WithSQLFile registers a SQL file to run after Postgres is ready.
*/
func (p *Postgres) WithSQLFile(path string) *Postgres {
	p.preloads = append(p.preloads, func(db *sql.DB) error {
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		_, err = db.Exec(string(contents))
		return err
	})

	return p
}

/*
Preload runs all registered SQL preload functions.
*/
func (p *Postgres) Preload() error {
	if len(p.preloads) == 0 {
		return nil
	}

	db, err := p.ConnectWithTimeout(10 * time.Second)
	if err != nil {
		return err
	}

	for _, preload := range p.preloads {
		err := preload(db)
		if err != nil {
			return fmt.Errorf("failed to preload postgres: %w", err)
		}
	}

	return nil
}

/*
Cleanup closes the database connection and removes the container.
*/
func (p *Postgres) Cleanup(ctx context.Context) error {
	if p.db != nil {
		p.db.Close()
		p.db = nil
	}

	return p.container.Cleanup(ctx)
}

/*
Logs returns the Postgres container logs keyed by service name.
*/
func (p *Postgres) Logs(ctx context.Context) (logs.LogStreams, error) {
	stream, err := p.container.Logs(ctx)
	if err != nil {
		return nil, err
	}

	return logs.LogStreams{p.name: stream}, nil
}
