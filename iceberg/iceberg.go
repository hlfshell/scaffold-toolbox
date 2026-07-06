package iceberg

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hlfshell/scaffold"
	minio "github.com/hlfshell/scaffold-toolbox/minio"
	trino "github.com/hlfshell/scaffold-toolbox/trino"
	scaffoldcontainer "github.com/hlfshell/scaffold/container"
	"github.com/hlfshell/scaffold/logs"
)

/*
Stack is a local Iceberg lakehouse environment. It composes MinIO for
object storage, an Iceberg REST catalog, and Trino for SQL queries.
*/
type Stack struct {
	Stack       *scaffold.Stack
	name        string
	Warehouse   *minio.MinIO
	Catalog     *Catalog
	Trino       *trino.Trino
	bucket      string
	accessKey   string
	secretKey   string
	catalogName string
}

// Option configures the Iceberg stack before the underlying services are built.
type Option func(*config)

type config struct {
	bucket      string
	accessKey   string
	secretKey   string
	catalogName string
}

/*
WithBucket sets the MinIO bucket used as the Iceberg warehouse.
*/
func WithBucket(bucket string) Option {
	return func(config *config) {
		if bucket != "" {
			config.bucket = bucket
		}
	}
}

/*
NewStack creates a local Iceberg stack using MinIO, the Iceberg REST
catalog image, and Trino with an Iceberg catalog.
*/
func NewStack(name string, options ...Option) (*Stack, error) {
	config := &config{
		bucket:      "warehouse",
		accessKey:   "minioadmin",
		secretKey:   "minioadmin",
		catalogName: "iceberg",
	}
	for _, option := range options {
		option(config)
	}

	warehouse, err := minio.NewMinIO(name+"-minio", "latest", config.accessKey, config.secretKey)
	if err != nil {
		return nil, err
	}
	warehouse.WithBucket(config.bucket)

	catalog, err := NewCatalog(name+"-catalog", "latest", config.bucket, config.accessKey, config.secretKey)
	if err != nil {
		return nil, err
	}

	query, err := trino.NewTrino(name+"-trino", "latest", trino.WithCatalog(config.catalogName, map[string]string{
		"connector.name":                 "iceberg",
		"iceberg.catalog.type":           "rest",
		"iceberg.rest-catalog.uri":       "http://" + name + "-catalog:8181",
		"iceberg.rest-catalog.warehouse": "s3://" + config.bucket + "/",
		"fs.native-s3.enabled":           "true",
		"s3.endpoint":                    "http://" + name + "-minio:9000",
		"s3.aws-access-key":              config.accessKey,
		"s3.aws-secret-key":              config.secretKey,
		"s3.path-style-access":           "true",
	}))
	if err != nil {
		return nil, err
	}

	stack := &Stack{
		name:        name,
		Warehouse:   warehouse,
		Catalog:     catalog,
		Trino:       query,
		bucket:      config.bucket,
		accessKey:   config.accessKey,
		secretKey:   config.secretKey,
		catalogName: config.catalogName,
	}
	stack.Stack = scaffold.NewStack(
		name,
		scaffold.WithServices(stack.Warehouse, stack.Catalog),
		scaffold.WithServices(stack.Trino),
		scaffold.WithSharedNetwork(),
	)

	return stack, nil
}

func (s *Stack) Name() string {
	return s.name
}

func (s *Stack) SetLabels(labels map[string]string) {
	s.Stack.SetLabels(labels)
}

func (s *Stack) SetNamePrefix(prefix string) {
	s.Stack.SetNamePrefix(prefix)
}

func (s *Stack) Create(ctx context.Context) error {
	return s.Stack.Create(ctx)
}

func (s *Stack) Cleanup(ctx context.Context) error {
	return s.Stack.Cleanup(ctx)
}

func (s *Stack) Env() map[string]string {
	env := s.Stack.Env()
	env["ICEBERG_BUCKET"] = s.bucket
	env["ICEBERG_CATALOG"] = s.catalogName
	return env
}

func (s *Stack) Endpoints() map[string]string {
	return s.Stack.Endpoints()
}

func (s *Stack) Logs(ctx context.Context) (logs.LogStreams, error) {
	return s.Stack.Logs(ctx)
}

/*
Catalog is a typed harness around the Iceberg REST catalog container.
*/
type Catalog struct {
	container *scaffoldcontainer.Container
	name      string
	port      string
}

/*
NewCatalog creates an Iceberg REST catalog pointed at the MinIO warehouse
service inside the shared scaffold network.
*/
func NewCatalog(name string, tag string, bucket string, accessKey string, secretKey string) (*Catalog, error) {
	container, err := scaffoldcontainer.NewContainer(
		name,
		"tabulario/iceberg-rest",
		scaffoldcontainer.WithTag(tag),
		scaffoldcontainer.WithPort("8181", ""),
		scaffoldcontainer.WithEnv(map[string]string{
			"AWS_ACCESS_KEY_ID":     accessKey,
			"AWS_SECRET_ACCESS_KEY": secretKey,
			"AWS_REGION":            "us-east-1",
			"CATALOG_WAREHOUSE":     "s3://" + bucket + "/",
			"CATALOG_IO__IMPL":      "org.apache.iceberg.aws.s3.S3FileIO",
			"CATALOG_S3_ENDPOINT":   "http://" + stringsTrimSuffix(name, "-catalog") + "-minio:9000",
		}),
	)
	if err != nil {
		return nil, err
	}

	return &Catalog{container: container, name: name}, nil
}

func (c *Catalog) Name() string {
	return c.name
}

func (c *Catalog) SetNetwork(name string) {
	c.container.SetNetwork(name)
}

func (c *Catalog) SetLabels(labels map[string]string) {
	c.container.SetLabels(labels)
}

func (c *Catalog) SetNamePrefix(prefix string) {
	c.container.SetNamePrefix(prefix)
}

func (c *Catalog) Create(ctx context.Context) error {
	err := c.container.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start iceberg catalog container: %w", err)
	}

	ports := c.container.GetPorts()
	c.port = ports["8181"]

	err = scaffold.WaitForHTTP(ctx, c.Endpoint()+"/v1/config", http.StatusOK, 60*time.Second)
	if err != nil {
		c.container.Cleanup(context.WithoutCancel(ctx))
		return fmt.Errorf("iceberg catalog failed to become ready: %w", err)
	}

	return nil
}

func (c *Catalog) Endpoint() string {
	return fmt.Sprintf("http://127.0.0.1:%s", c.port)
}

func (c *Catalog) Env() map[string]string {
	return map[string]string{
		"ICEBERG_REST_URL": c.Endpoint(),
	}
}

func (c *Catalog) Endpoints() map[string]string {
	return map[string]string{
		c.name: c.Endpoint(),
	}
}

func (c *Catalog) Cleanup(ctx context.Context) error {
	return c.container.Cleanup(ctx)
}

func (c *Catalog) Logs(ctx context.Context) (logs.LogStreams, error) {
	stream, err := c.container.Logs(ctx)
	if err != nil {
		return nil, err
	}

	return logs.LogStreams{c.name: stream}, nil
}

func stringsTrimSuffix(value string, suffix string) string {
	if len(value) >= len(suffix) && value[len(value)-len(suffix):] == suffix {
		return value[:len(value)-len(suffix)]
	}

	return value
}

var _ scaffold.Service = (*Stack)(nil)
var _ scaffold.Service = (*Catalog)(nil)
