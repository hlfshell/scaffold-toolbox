package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	driver "github.com/go-sql-driver/mysql"
	"github.com/hlfshell/scaffold"
	scaffoldcontainer "github.com/hlfshell/scaffold/container"
	"github.com/hlfshell/scaffold/logs"
)

func init() {
	// Suppress MySQL driver error logging during connection retries.
	driver.SetLogger(&driver.NopLogger{})
}

/*
Mysql is a typed harness around the official mysql container. It keeps
the assigned port, connection settings, and preload functions.
*/
type Mysql struct {
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
NewMysql creates a MySQL harness and configures the correct environment
variables for either root or regular users.
*/
func NewMysql(name string, tag string, username string, password string, database string) (*Mysql, error) {
	// Root user cannot be configured through MYSQL_USER, so it needs a
	// slightly different environment setup.
	env := make(map[string]string)
	if username == "root" {
		env["MYSQL_ROOT_PASSWORD"] = password
		env["MYSQL_DATABASE"] = database
	} else {
		env["MYSQL_USER"] = username
		env["MYSQL_PASSWORD"] = password
		env["MYSQL_ROOT_PASSWORD"] = password
		env["MYSQL_DATABASE"] = database
	}

	container, err := scaffoldcontainer.NewContainer(
		name,
		"mysql",
		scaffoldcontainer.WithTag(tag),
		scaffoldcontainer.WithPort("3306", ""),
		scaffoldcontainer.WithEnv(env),
	)
	if err != nil {
		return nil, err
	}

	return &Mysql{
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
func (m *Mysql) Name() string {
	return m.name
}

/*
SetNetwork attaches the underlying container to a Docker network when it
is created.
*/
func (m *Mysql) SetNetwork(name string) {
	m.container.SetNetwork(name)
}

/*
SetLabels merges Docker labels onto the underlying container.
*/
func (m *Mysql) SetLabels(labels map[string]string) {
	m.container.SetLabels(labels)
}

/*
SetNamePrefix prefixes the underlying Docker container name before it is
created.
*/
func (m *Mysql) SetNamePrefix(prefix string) {
	m.container.SetNamePrefix(prefix)
}

/*
Create starts MySQL with ctx, waits until it accepts connections, and runs any
registered preload functions.
*/
func (m *Mysql) Create(ctx context.Context) error {
	err := m.container.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start mysql container: %w", err)
	}

	ports := m.container.GetPorts()
	m.port = ports["3306"]

	err = scaffold.WaitFunc(ctx, 60*time.Second, 50*time.Millisecond, func(ctx context.Context) error {
		db, err := m.connectContext(ctx)
		if err != nil {
			return err
		}

		return db.PingContext(ctx)
	})
	if err != nil {
		m.container.Cleanup(context.WithoutCancel(ctx))
		return fmt.Errorf("mysql failed to become ready: %w", err)
	}

	err = m.Preload(ctx)
	if err != nil {
		m.container.Cleanup(context.WithoutCancel(ctx))
		return err
	}

	return nil
}

/*
Connect opens and pings a database connection using the assigned host
port.
*/
func (m *Mysql) Connect(ctx context.Context) (*sql.DB, error) {
	return m.connectContext(ctx)
}

func (m *Mysql) connectContext(ctx context.Context) (*sql.DB, error) {
	db, err := sql.Open("mysql", m.ConnectionString())
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	err = db.PingContext(ctx)
	if err != nil {
		db.Close()
		return nil, err
	}

	if m.db != nil {
		m.db.Close()
	}
	m.db = db

	return db, nil
}

/*
ConnectWithTimeout repeatedly calls Connect until a connection succeeds
or the timeout is reached.
*/
func (m *Mysql) ConnectWithTimeout(ctx context.Context, timeout time.Duration) (*sql.DB, error) {
	var db *sql.DB

	err := scaffold.WaitFunc(ctx, timeout, 50*time.Millisecond, func(ctx context.Context) error {
		var err error
		db, err = m.connectContext(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}

	return db, nil
}

/*
ConnectionString returns the local MySQL connection string.
*/
func (m *Mysql) ConnectionString() string {
	return fmt.Sprintf("%s:%s@tcp(127.0.0.1:%s)/%s", m.username, m.password, m.port, m.database)
}

/*
Env returns MySQL connection environment variables.
*/
func (m *Mysql) Env() map[string]string {
	return map[string]string{
		"MYSQL_DSN": m.ConnectionString(),
	}
}

/*
Endpoints returns named MySQL endpoints.
*/
func (m *Mysql) Endpoints() map[string]string {
	return map[string]string{
		m.name: "127.0.0.1:" + m.port,
	}
}

/*
WithSQL registers a SQL string to run after MySQL is ready.
*/
func (m *Mysql) WithSQL(query string) *Mysql {
	m.preloads = append(m.preloads, func(db *sql.DB) error {
		_, err := db.Exec(query)
		return err
	})

	return m
}

/*
WithSQLFile registers a SQL file to run after MySQL is ready.
*/
func (m *Mysql) WithSQLFile(path string) *Mysql {
	m.preloads = append(m.preloads, func(db *sql.DB) error {
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		_, err = db.Exec(string(contents))
		return err
	})

	return m
}

/*
Preload runs all registered SQL preload functions.
*/
func (m *Mysql) Preload(ctx context.Context) error {
	if len(m.preloads) == 0 {
		return nil
	}

	db, err := m.ConnectWithTimeout(ctx, 10*time.Second)
	if err != nil {
		return err
	}

	for _, preload := range m.preloads {
		err := preload(db)
		if err != nil {
			return fmt.Errorf("failed to preload mysql: %w", err)
		}
	}

	return nil
}

/*
GetDB returns the last successful database connection.
*/
func (m *Mysql) GetDB() *sql.DB {
	return m.db
}

/*
GetContainer returns the underlying Scaffold container.
*/
func (m *Mysql) GetContainer() *scaffoldcontainer.Container {
	return m.container
}

/*
Cleanup closes the database connection and removes the container.
*/
func (m *Mysql) Cleanup(ctx context.Context) error {
	if m.db != nil {
		m.db.Close()
		m.db = nil
	}

	return m.container.Cleanup(ctx)
}

/*
Logs returns the MySQL container logs keyed by service name.
*/
func (m *Mysql) Logs(ctx context.Context) (logs.LogStreams, error) {
	stream, err := m.container.Logs(ctx)
	if err != nil {
		return nil, err
	}

	return logs.LogStreams{m.name: stream}, nil
}
