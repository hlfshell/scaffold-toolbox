package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/hlfshell/scaffold"
	scaffoldcontainer "github.com/hlfshell/scaffold/container"
	"github.com/hlfshell/scaffold/logs"
	goredis "github.com/redis/go-redis/v9"
)

/*
Redis is a typed harness around the official redis container. It keeps
the assigned port, client, and preload functions.
*/
type Redis struct {
	container *scaffoldcontainer.Container
	client    *goredis.Client
	name      string
	port      string
	preloads  []func(context.Context, *goredis.Client) error
}

/*
NewRedis creates a Redis harness using the default redis image tag.
*/
func NewRedis(name string) (*Redis, error) {
	container, err := scaffoldcontainer.NewContainer(
		name,
		"redis",
		scaffoldcontainer.WithPort("6379", ""),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create redis container: %w", err)
	}

	return &Redis{
		container: container,
		name:      name,
		preloads:  []func(context.Context, *goredis.Client) error{},
	}, nil
}

/*
Name returns the service name used by Scaffold stacks.
*/
func (r *Redis) Name() string {
	return r.name
}

/*
SetNetwork attaches the underlying container to a Docker network when it
is created.
*/
func (r *Redis) SetNetwork(name string) {
	r.container.SetNetwork(name)
}

/*
SetLabels merges Docker labels onto the underlying container.
*/
func (r *Redis) SetLabels(labels map[string]string) {
	r.container.SetLabels(labels)
}

/*
SetNamePrefix prefixes the underlying Docker container name before it is
created.
*/
func (r *Redis) SetNamePrefix(prefix string) {
	r.container.SetNamePrefix(prefix)
}

/*
Create starts Redis, waits until it responds to PING, and runs any registered
preload functions.
*/
func (r *Redis) Create(ctx context.Context) error {
	err := r.container.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start redis container: %w", err)
	}

	ports := r.container.GetPorts()
	r.port = ports["6379"]

	_, err = r.connectWithTimeoutContext(ctx, 10*time.Second)
	if err != nil {
		r.container.Cleanup(context.WithoutCancel(ctx))
		return err
	}

	err = r.preloadContext(ctx)
	if err != nil {
		r.container.Cleanup(context.WithoutCancel(ctx))
		return err
	}

	return nil
}

/*
Connect creates a Redis client and verifies it with PING.
*/
func (r *Redis) Connect() (*goredis.Client, error) {
	return r.connectContext(context.Background())
}

func (r *Redis) connectContext(ctx context.Context) (*goredis.Client, error) {
	client := goredis.NewClient(&goredis.Options{
		Addr: fmt.Sprintf("localhost:%s", r.port),
		DB:   0,
	})

	pong, err := client.Ping(ctx).Result()
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	} else if pong != "PONG" {
		client.Close()
		return nil, fmt.Errorf("unexpected ping response: %s", pong)
	}

	if r.client != nil {
		r.client.Close()
	}
	r.client = client

	return client, nil
}

/*
ConnectWithTimeout repeatedly calls Connect until a client is ready or
the timeout is reached.
*/
func (r *Redis) ConnectWithTimeout(timeout time.Duration) (*goredis.Client, error) {
	return r.connectWithTimeoutContext(context.Background(), timeout)
}

func (r *Redis) connectWithTimeoutContext(ctx context.Context, timeout time.Duration) (*goredis.Client, error) {
	var client *goredis.Client

	err := scaffold.WaitFunc(ctx, timeout, 50*time.Millisecond, func(ctx context.Context) error {
		var err error
		client, err = r.connectContext(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}

	return client, nil
}

/*
Env returns Redis connection environment variables.
*/
func (r *Redis) Env() map[string]string {
	return map[string]string{
		"REDIS_URL": fmt.Sprintf("redis://localhost:%s/0", r.port),
	}
}

/*
Endpoints returns named Redis endpoints.
*/
func (r *Redis) Endpoints() map[string]string {
	return map[string]string{
		r.name: fmt.Sprintf("localhost:%s", r.port),
	}
}

/*
WithKey registers a key/value pair to set after Redis is ready.
*/
func (r *Redis) WithKey(key string, value string) *Redis {
	r.preloads = append(r.preloads, func(ctx context.Context, client *goredis.Client) error {
		return client.Set(ctx, key, value, 0).Err()
	})

	return r
}

/*
WithSeed registers a custom Redis seed function.
*/
func (r *Redis) WithSeed(fn func(context.Context, *goredis.Client) error) *Redis {
	r.preloads = append(r.preloads, fn)
	return r
}

/*
Preload runs all registered Redis seed functions.
*/
func (r *Redis) Preload() error {
	return r.preloadContext(context.Background())
}

func (r *Redis) preloadContext(ctx context.Context) error {
	if len(r.preloads) == 0 {
		return nil
	}

	client, err := r.connectWithTimeoutContext(ctx, 10*time.Second)
	if err != nil {
		return err
	}

	for _, preload := range r.preloads {
		err := preload(ctx, client)
		if err != nil {
			return fmt.Errorf("failed to preload redis: %w", err)
		}
	}

	return nil
}

/*
GetClient returns the last successful Redis client.
*/
func (r *Redis) GetClient() *goredis.Client {
	return r.client
}

/*
Cleanup closes the Redis client and removes the container.
*/
func (r *Redis) Cleanup(ctx context.Context) error {
	if r.client != nil {
		r.client.Close()
		r.client = nil
	}

	return r.container.Cleanup(ctx)
}

/*
Logs returns the Redis container logs keyed by service name.
*/
func (r *Redis) Logs(ctx context.Context) (logs.LogStreams, error) {
	stream, err := r.container.Logs(ctx)
	if err != nil {
		return nil, err
	}

	return logs.LogStreams{r.name: stream}, nil
}
