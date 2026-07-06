package clickhouse

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/hlfshell/scaffold"
	scaffoldcontainer "github.com/hlfshell/scaffold/container"
	"github.com/hlfshell/scaffold/logs"
)

/*
ClickHouse is a typed harness around clickhouse/clickhouse-server. It
exposes HTTP and native endpoints and runs SQL preload statements.
*/
type ClickHouse struct {
	container  *scaffoldcontainer.Container
	client     *http.Client
	name       string
	username   string
	password   string
	database   string
	httpPort   string
	nativePort string
	preloads   []func(context.Context, *ClickHouse) error
}

/*
NewClickHouse creates a ClickHouse service using the official server
image.
*/
func NewClickHouse(name string, tag string, username string, password string, database string) (*ClickHouse, error) {
	container, err := scaffoldcontainer.NewContainer(
		name,
		"clickhouse/clickhouse-server",
		scaffoldcontainer.WithTag(tag),
		scaffoldcontainer.WithPorts(map[string]string{
			"8123": "",
			"9000": "",
		}),
		scaffoldcontainer.WithEnv(map[string]string{
			"CLICKHOUSE_USER":                      username,
			"CLICKHOUSE_PASSWORD":                  password,
			"CLICKHOUSE_DB":                        database,
			"CLICKHOUSE_DEFAULT_ACCESS_MANAGEMENT": "1",
		}),
	)
	if err != nil {
		return nil, err
	}

	return &ClickHouse{
		container: container,
		client:    &http.Client{Timeout: 15 * time.Second},
		name:      name,
		username:  username,
		password:  password,
		database:  database,
		preloads:  []func(context.Context, *ClickHouse) error{},
	}, nil
}

func (c *ClickHouse) Name() string {
	return c.name
}

/*
SetNetwork attaches the underlying container to a shared Docker network.
*/
func (c *ClickHouse) SetNetwork(name string) {
	c.container.SetNetwork(name)
}

/*
SetLabels merges inherited Docker labels onto the container.
*/
func (c *ClickHouse) SetLabels(labels map[string]string) {
	c.container.SetLabels(labels)
}

/*
SetNamePrefix prefixes the Docker container name before it is created.
*/
func (c *ClickHouse) SetNamePrefix(prefix string) {
	c.container.SetNamePrefix(prefix)
}

/*
Create starts ClickHouse, waits for the HTTP ping endpoint, and runs
registered SQL preload statements.
*/
func (c *ClickHouse) Create(ctx context.Context) error {
	err := c.container.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start clickhouse container: %w", err)
	}

	ports := c.container.GetPorts()
	c.httpPort = ports["8123"]
	c.nativePort = ports["9000"]

	err = scaffold.WaitForHTTP(ctx, c.HTTPURL()+"/ping", http.StatusOK, 60*time.Second)
	if err != nil {
		c.container.Cleanup(context.WithoutCancel(ctx))
		return fmt.Errorf("clickhouse failed to become ready: %w", err)
	}

	err = c.Preload(ctx)
	if err != nil {
		c.container.Cleanup(context.WithoutCancel(ctx))
		return err
	}

	return nil
}

/*
HTTPURL returns the local ClickHouse HTTP endpoint.
*/
func (c *ClickHouse) HTTPURL() string {
	return fmt.Sprintf("http://127.0.0.1:%s", c.httpPort)
}

/*
NativeAddress returns the local ClickHouse native protocol address.
*/
func (c *ClickHouse) NativeAddress() string {
	return fmt.Sprintf("127.0.0.1:%s", c.nativePort)
}

func (c *ClickHouse) Env() map[string]string {
	return map[string]string{
		"CLICKHOUSE_HTTP_URL": c.HTTPURL(),
		"CLICKHOUSE_NATIVE":   c.NativeAddress(),
		"CLICKHOUSE_USER":     c.username,
		"CLICKHOUSE_PASSWORD": c.password,
		"CLICKHOUSE_DATABASE": c.database,
	}
}

func (c *ClickHouse) Endpoints() map[string]string {
	return map[string]string{
		c.name:             c.HTTPURL(),
		c.name + "-native": c.NativeAddress(),
	}
}

/*
Exec runs a SQL statement through the ClickHouse HTTP interface.
*/
func (c *ClickHouse) Exec(ctx context.Context, sql string) error {
	endpoint := c.HTTPURL() + "/"
	query := url.Values{}
	if c.database != "" {
		query.Set("database", c.database)
	}
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(sql))
	if err != nil {
		return err
	}
	if c.username != "" || c.password != "" {
		request.SetBasicAuth(c.username, c.password)
	}

	response, err := c.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("clickhouse query failed with status %d", response.StatusCode)
	}

	return nil
}

/*
WithSQL registers SQL to run after ClickHouse is ready.
*/
func (c *ClickHouse) WithSQL(sql string) *ClickHouse {
	c.preloads = append(c.preloads, func(ctx context.Context, clickhouse *ClickHouse) error {
		return clickhouse.Exec(ctx, sql)
	})

	return c
}

/*
WithSQLFile registers a SQL file to run after ClickHouse is ready.
*/
func (c *ClickHouse) WithSQLFile(path string) *ClickHouse {
	c.preloads = append(c.preloads, func(ctx context.Context, clickhouse *ClickHouse) error {
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return clickhouse.Exec(ctx, string(contents))
	})

	return c
}

/*
Preload runs all registered SQL preload functions.
*/
func (c *ClickHouse) Preload(ctx context.Context) error {
	for _, preload := range c.preloads {
		err := preload(ctx, c)
		if err != nil {
			return fmt.Errorf("failed to preload clickhouse: %w", err)
		}
	}

	return nil
}

/*
Cleanup removes the ClickHouse container.
*/
func (c *ClickHouse) Cleanup(ctx context.Context) error {
	return c.container.Cleanup(ctx)
}

/*
Logs returns ClickHouse container logs keyed by service name.
*/
func (c *ClickHouse) Logs(ctx context.Context) (logs.LogStreams, error) {
	stream, err := c.container.Logs(ctx)
	if err != nil {
		return nil, err
	}

	return logs.LogStreams{c.name: stream}, nil
}
