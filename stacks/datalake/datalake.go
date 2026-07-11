package datalake

import (
	"context"

	"github.com/hlfshell/scaffold"
	iceberg "github.com/hlfshell/scaffold-toolbox/iceberg"
	"github.com/hlfshell/scaffold/logs"
)

/*
Stack is a local lakehouse environment backed by MinIO, an
Iceberg REST catalog, and Trino.
*/
type Stack struct {
	Lake *iceberg.Stack
	name string
}

// Option configures the data lake stack.
type Option func(*dataLakeConfig)

type dataLakeConfig struct {
	icebergOptions []iceberg.Option
}

/*
WithBucket sets the object storage bucket used as the Iceberg
warehouse.
*/
func WithBucket(bucket string) Option {
	return func(config *dataLakeConfig) {
		config.icebergOptions = append(config.icebergOptions, iceberg.WithBucket(bucket))
	}
}

/*
WithIcebergOptions passes options directly to the underlying Iceberg
stack.
*/
func WithIcebergOptions(options ...iceberg.Option) Option {
	return func(config *dataLakeConfig) {
		config.icebergOptions = append(config.icebergOptions, options...)
	}
}

/*
NewStack creates a local Iceberg lakehouse stack.
*/
func NewStack(name string, options ...Option) (*Stack, error) {
	config := &dataLakeConfig{}
	for _, option := range options {
		option(config)
	}

	lake, err := iceberg.NewStack(name+"-lake", config.icebergOptions...)
	if err != nil {
		return nil, err
	}

	return &Stack{name: name, Lake: lake}, nil
}

func (d *Stack) Name() string {
	return d.name
}

/*
SetLabels passes inherited labels to the underlying Iceberg stack.
*/
func (d *Stack) SetLabels(labels map[string]string) {
	d.Lake.SetLabels(labels)
}

/*
SetNamePrefix passes an inherited Docker name prefix to the underlying
Iceberg stack.
*/
func (d *Stack) SetNamePrefix(prefix string) {
	d.Lake.SetNamePrefix(prefix)
}

/*
Create starts MinIO, the Iceberg REST catalog, and Trino.
*/
func (d *Stack) Create(ctx context.Context) error {
	return d.Lake.Create(ctx)
}

/*
Env returns environment variables exposed by the data lake stack.
*/
func (d *Stack) Env() map[string]string {
	return d.Lake.Env()
}

/*
Endpoints returns endpoints exposed by the data lake stack.
*/
func (d *Stack) Endpoints() map[string]string {
	return d.Lake.Endpoints()
}

/*
Query runs SQL through the data lake Trino service.
*/
func (d *Stack) Query(ctx context.Context, sql string) ([]byte, error) {
	return d.Lake.Trino.Query(ctx, sql)
}

/*
Cleanup removes resources created by the data lake stack.
*/
func (d *Stack) Cleanup(ctx context.Context) error {
	return d.Lake.Cleanup(ctx)
}

/*
Logs returns logs from the data lake services.
*/
func (d *Stack) Logs(ctx context.Context) (logs.LogStreams, error) {
	return d.Lake.Logs(ctx)
}

var _ scaffold.Service = (*Stack)(nil)
var _ scaffold.LabelAttachable = (*Stack)(nil)
var _ scaffold.NamePrefixAttachable = (*Stack)(nil)
