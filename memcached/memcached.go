package memcached

import (
	"context"
	"fmt"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/hlfshell/scaffold"
	scaffoldcontainer "github.com/hlfshell/scaffold/container"
	"github.com/hlfshell/scaffold/logs"
)

/*
Memcached is a typed harness around the official memcached container. It
keeps the assigned port, client, and preload items.
*/
type Memcached struct {
	container *scaffoldcontainer.Container
	client    *memcache.Client
	name      string
	port      string
	preloads  []*memcache.Item
}

/*
NewMemcached creates a Memcached harness using the default memcached
image tag.
*/
func NewMemcached(name string) (*Memcached, error) {
	container, err := scaffoldcontainer.NewContainer(
		name,
		"memcached",
		scaffoldcontainer.WithPort("11211", ""),
	)
	if err != nil {
		return nil, err
	}

	return &Memcached{
		container: container,
		name:      name,
		preloads:  []*memcache.Item{},
	}, nil
}

/*
Name returns the service name used by Scaffold stacks.
*/
func (m *Memcached) Name() string {
	return m.name
}

/*
SetNetwork attaches the underlying container to a Docker network when it
is created.
*/
func (m *Memcached) SetNetwork(name string) {
	m.container.SetNetwork(name)
}

/*
SetLabels merges Docker labels onto the underlying container.
*/
func (m *Memcached) SetLabels(labels map[string]string) {
	m.container.SetLabels(labels)
}

/*
SetNamePrefix prefixes the underlying Docker container name before it is
created.
*/
func (m *Memcached) SetNamePrefix(prefix string) {
	m.container.SetNamePrefix(prefix)
}

/*
Create starts Memcached with ctx, waits until it responds to ping,
and runs any registered preload items.
*/
func (m *Memcached) Create(ctx context.Context) error {
	err := m.container.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start memcached container: %w", err)
	}

	ports := m.container.GetPorts()
	m.port = ports["11211"]

	_, err = m.connectWithTimeoutContext(ctx, 10*time.Second)
	if err != nil {
		m.container.Cleanup(context.WithoutCancel(ctx))
		return err
	}

	err = m.Preload()
	if err != nil {
		m.container.Cleanup(context.WithoutCancel(ctx))
		return err
	}

	return nil
}

/*
Connect creates a Memcached client and verifies it with Ping.
*/
func (m *Memcached) Connect() (*memcache.Client, error) {
	client := memcache.New(fmt.Sprintf("127.0.0.1:%s", m.port))

	err := client.Ping()
	if err != nil {
		return nil, err
	}

	m.client = client
	return client, nil
}

/*
ConnectWithTimeout repeatedly calls Connect until a client is ready or
the timeout is reached.
*/
func (m *Memcached) ConnectWithTimeout(timeout time.Duration) (*memcache.Client, error) {
	return m.connectWithTimeoutContext(context.Background(), timeout)
}

func (m *Memcached) connectWithTimeoutContext(ctx context.Context, timeout time.Duration) (*memcache.Client, error) {
	var client *memcache.Client

	err := scaffold.WaitFunc(ctx, timeout, 50*time.Millisecond, func(ctx context.Context) error {
		var err error
		client, err = m.Connect()
		return err
	})
	if err != nil {
		return nil, err
	}

	return client, nil
}

/*
Env returns Memcached connection environment variables.
*/
func (m *Memcached) Env() map[string]string {
	return map[string]string{
		"MEMCACHED_ADDR": fmt.Sprintf("127.0.0.1:%s", m.port),
	}
}

/*
Endpoints returns named Memcached endpoints.
*/
func (m *Memcached) Endpoints() map[string]string {
	return map[string]string{
		m.name: fmt.Sprintf("127.0.0.1:%s", m.port),
	}
}

/*
WithItem registers a cache item to set after Memcached is ready.
*/
func (m *Memcached) WithItem(key string, value []byte) *Memcached {
	m.preloads = append(m.preloads, &memcache.Item{
		Key:   key,
		Value: value,
	})

	return m
}

/*
Preload writes all registered cache items.
*/
func (m *Memcached) Preload() error {
	if len(m.preloads) == 0 {
		return nil
	}

	client, err := m.ConnectWithTimeout(10 * time.Second)
	if err != nil {
		return err
	}

	for _, item := range m.preloads {
		err := client.Set(item)
		if err != nil {
			return fmt.Errorf("failed to preload memcached: %w", err)
		}
	}

	return nil
}

/*
GetClient returns the last successful Memcached client.
*/
func (m *Memcached) GetClient() *memcache.Client {
	return m.client
}

/*
Cleanup closes the Memcached client and removes the container.
*/
func (m *Memcached) Cleanup(ctx context.Context) error {
	if m.client != nil {
		m.client.Close()
		m.client = nil
	}

	return m.container.Cleanup(ctx)
}

/*
Logs returns the Memcached container logs keyed by service name.
*/
func (m *Memcached) Logs(ctx context.Context) (logs.LogStreams, error) {
	stream, err := m.container.Logs(ctx)
	if err != nil {
		return nil, err
	}

	return logs.LogStreams{m.name: stream}, nil
}
