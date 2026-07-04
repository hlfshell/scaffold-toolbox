package minio

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hlfshell/scaffold"
	scaffoldcontainer "github.com/hlfshell/scaffold/container"
	"github.com/hlfshell/scaffold/logs"
	minioclient "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

/*
MinIO is a typed harness around the minio/minio container. It exposes the
S3-compatible API endpoint, console endpoint, and preload helpers.
*/
type MinIO struct {
	container *scaffoldcontainer.Container
	client    *minioclient.Client
	name      string
	accessKey string
	secretKey string
	apiPort   string
	webPort   string
	preloads  []func(context.Context, *MinIO) error
}

/*
NewMinIO creates a MinIO harness with root credentials and the command
required to start the object store.
*/
func NewMinIO(name string, tag string, accessKey string, secretKey string) (*MinIO, error) {
	container, err := scaffoldcontainer.NewContainer(
		name,
		"minio/minio",
		scaffoldcontainer.WithTag(tag),
		scaffoldcontainer.WithPorts(map[string]string{
			"9000": "",
			"9001": "",
		}),
		scaffoldcontainer.WithEnv(map[string]string{
			"MINIO_ROOT_USER":     accessKey,
			"MINIO_ROOT_PASSWORD": secretKey,
		}),
		scaffoldcontainer.WithCommand("server", "/data", "--console-address", ":9001"),
	)
	if err != nil {
		return nil, err
	}

	return &MinIO{
		container: container,
		name:      name,
		accessKey: accessKey,
		secretKey: secretKey,
		preloads:  []func(context.Context, *MinIO) error{},
	}, nil
}

/*
Name returns the service name used by Scaffold stacks.
*/
func (m *MinIO) Name() string {
	return m.name
}

/*
SetNetwork attaches the underlying container to a Docker network when it
is created.
*/
func (m *MinIO) SetNetwork(name string) {
	m.container.SetNetwork(name)
}

/*
SetLabels merges Docker labels onto the underlying container.
*/
func (m *MinIO) SetLabels(labels map[string]string) {
	m.container.SetLabels(labels)
}

/*
SetNamePrefix prefixes the underlying Docker container name before it is
created.
*/
func (m *MinIO) SetNamePrefix(prefix string) {
	m.container.SetNamePrefix(prefix)
}

/*
Create starts MinIO with ctx, waits for the health endpoint,
creates a client, and runs any registered preload functions.
*/
func (m *MinIO) Create(ctx context.Context) error {
	err := m.container.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start minio container: %w", err)
	}

	ports := m.container.GetPorts()
	m.apiPort = ports["9000"]
	m.webPort = ports["9001"]

	err = scaffold.WaitForHTTP(ctx, m.Endpoint()+"/minio/health/ready", httpStatusOK, 30*time.Second)
	if err != nil {
		m.container.Cleanup(context.WithoutCancel(ctx))
		return fmt.Errorf("minio failed to become ready: %w", err)
	}

	_, err = m.Client()
	if err != nil {
		m.container.Cleanup(context.WithoutCancel(ctx))
		return err
	}

	err = scaffold.WaitFunc(ctx, 30*time.Second, 250*time.Millisecond, func(ctx context.Context) error {
		return m.Preload(ctx)
	})
	if err != nil {
		m.container.Cleanup(context.WithoutCancel(ctx))
		return err
	}

	return nil
}

/*
Endpoint returns the local S3-compatible API endpoint.
*/
func (m *MinIO) Endpoint() string {
	return fmt.Sprintf("http://127.0.0.1:%s", m.apiPort)
}

/*
ConsoleEndpoint returns the local MinIO web console endpoint.
*/
func (m *MinIO) ConsoleEndpoint() string {
	return fmt.Sprintf("http://127.0.0.1:%s", m.webPort)
}

/*
Env returns MinIO endpoint and credential environment variables.
*/
func (m *MinIO) Env() map[string]string {
	return map[string]string{
		"MINIO_ENDPOINT":   m.Endpoint(),
		"MINIO_CONSOLE":    m.ConsoleEndpoint(),
		"MINIO_ACCESS_KEY": m.accessKey,
		"MINIO_SECRET_KEY": m.secretKey,
	}
}

/*
Endpoints returns named MinIO endpoints.
*/
func (m *MinIO) Endpoints() map[string]string {
	return map[string]string{
		m.name:              m.Endpoint(),
		m.name + "-console": m.ConsoleEndpoint(),
	}
}

/*
Client creates a MinIO client using the assigned host port and configured
root credentials.
*/
func (m *MinIO) Client() (*minioclient.Client, error) {
	client, err := minioclient.New(fmt.Sprintf("127.0.0.1:%s", m.apiPort), &minioclient.Options{
		Creds:  credentials.NewStaticV4(m.accessKey, m.secretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, err
	}

	m.client = client
	return client, nil
}

/*
CreateBucket creates a bucket if it does not already exist.
*/
func (m *MinIO) CreateBucket(ctx context.Context, bucket string) error {
	client, err := m.Client()
	if err != nil {
		return err
	}

	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	return client.MakeBucket(ctx, bucket, minioclient.MakeBucketOptions{})
}

/*
UploadBytes creates the bucket if needed and uploads an object from a
byte slice.
*/
func (m *MinIO) UploadBytes(ctx context.Context, bucket string, object string, contents []byte, contentType string) error {
	err := m.CreateBucket(ctx, bucket)
	if err != nil {
		return err
	}

	client, err := m.Client()
	if err != nil {
		return err
	}

	_, err = client.PutObject(ctx, bucket, object, bytes.NewReader(contents), int64(len(contents)), minioclient.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

/*
UploadFile creates the bucket if needed and uploads an object from a
local file.
*/
func (m *MinIO) UploadFile(ctx context.Context, bucket string, object string, path string, contentType string) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return m.UploadBytes(ctx, bucket, object, contents, contentType)
}

/*
WithBucket registers a bucket to create after MinIO is ready.
*/
func (m *MinIO) WithBucket(bucket string) *MinIO {
	m.preloads = append(m.preloads, func(ctx context.Context, minio *MinIO) error {
		return minio.CreateBucket(ctx, bucket)
	})

	return m
}

/*
WithObject registers an object to upload after MinIO is ready.
*/
func (m *MinIO) WithObject(bucket string, object string, contents []byte, contentType string) *MinIO {
	m.preloads = append(m.preloads, func(ctx context.Context, minio *MinIO) error {
		return minio.UploadBytes(ctx, bucket, object, contents, contentType)
	})

	return m
}

/*
Preload runs all registered MinIO preload functions.
*/
func (m *MinIO) Preload(ctx context.Context) error {
	for _, preload := range m.preloads {
		err := preload(ctx, m)
		if err != nil {
			return fmt.Errorf("failed to preload minio: %w", err)
		}
	}

	return nil
}

/*
Cleanup removes the MinIO container.
*/
func (m *MinIO) Cleanup(ctx context.Context) error {
	return m.container.Cleanup(ctx)
}

/*
Logs returns the MinIO container logs keyed by service name.
*/
func (m *MinIO) Logs(ctx context.Context) (logs.LogStreams, error) {
	stream, err := m.container.Logs(ctx)
	if err != nil {
		return nil, err
	}

	return logs.LogStreams{m.name: stream}, nil
}

const httpStatusOK = 200
