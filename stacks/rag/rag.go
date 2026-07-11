package rag

import (
	"context"

	"github.com/hlfshell/scaffold"
	minio "github.com/hlfshell/scaffold-toolbox/minio"
	postgres "github.com/hlfshell/scaffold-toolbox/postgres"
	qdrant "github.com/hlfshell/scaffold-toolbox/qdrant"
	"github.com/hlfshell/scaffold/logs"
)

/*
Stack is a ready-made stack for local retrieval-augmented generation
development. It composes Postgres, Qdrant, and MinIO.
*/
type Stack struct {
	Stack    *scaffold.Stack
	name     string
	Postgres *postgres.Postgres
	Qdrant   *qdrant.Qdrant
	MinIO    *minio.MinIO
}

// Option configures the RAG stack before the underlying Stack is built.
type Option func(*Stack)

/*
NewStack creates a local RAG development stack using the toolbox
harnesses. The stack does not duplicate service implementation.
*/
func NewStack(name string, options ...Option) (*Stack, error) {
	db, err := postgres.NewPostgres(name+"-postgres", "latest", "scaffold", "scaffold", "scaffold")
	if err != nil {
		return nil, err
	}

	vectors, err := qdrant.NewQdrant(name+"-qdrant", "latest")
	if err != nil {
		return nil, err
	}

	objects, err := minio.NewMinIO(name+"-minio", "latest", "minioadmin", "minioadmin")
	if err != nil {
		return nil, err
	}

	rag := &Stack{
		name:     name,
		Postgres: db,
		Qdrant:   vectors,
		MinIO:    objects,
	}

	for _, option := range options {
		option(rag)
	}

	rag.Stack = scaffold.NewStack(
		name,
		scaffold.WithServices(rag.Postgres, rag.Qdrant, rag.MinIO),
		scaffold.WithSharedNetwork(),
	)

	return rag, nil
}

/*
Name returns the stack name.
*/
func (r *Stack) Name() string {
	return r.name
}

/*
SetLabels passes inherited labels to the underlying stack.
*/
func (r *Stack) SetLabels(labels map[string]string) {
	r.Stack.SetLabels(labels)
}

/*
SetNamePrefix passes an inherited name prefix to the underlying stack.
*/
func (r *Stack) SetNamePrefix(prefix string) {
	r.Stack.SetNamePrefix(prefix)
}

/*
WithPostgresSQL registers SQL to preload into the Postgres service.
*/
func WithPostgresSQL(sql string) Option {
	return func(r *Stack) {
		r.Postgres.WithSQL(sql)
	}
}

/*
WithQdrantCollection registers a Qdrant collection to create during
preload.
*/
func WithQdrantCollection(config qdrant.CollectionConfig) Option {
	return func(r *Stack) {
		r.Qdrant.WithCollection(config)
	}
}

/*
WithMinIOBucket registers a MinIO bucket to create during preload.
*/
func WithMinIOBucket(bucket string) Option {
	return func(r *Stack) {
		r.MinIO.WithBucket(bucket)
	}
}

/*
Create creates the services in the underlying Scaffold stack with ctx.
*/
func (r *Stack) Create(ctx context.Context) error {
	return r.Stack.Create(ctx)
}

/*
IsRunning returns true if any matching container for this stack is
running.
*/
func (r *Stack) IsRunning(ctx context.Context) (bool, error) {
	return r.Stack.IsRunning(ctx)
}

/*
Resources returns matching Docker resources for this preset stack.
*/
func (r *Stack) Resources(ctx context.Context) (scaffold.ResourceStatus, error) {
	return r.Stack.Resources(ctx)
}

/*
Env returns environment variables from the underlying stack.
*/
func (r *Stack) Env() map[string]string {
	return r.Stack.Env()
}

/*
Endpoints returns endpoints from the underlying stack.
*/
func (r *Stack) Endpoints() map[string]string {
	return r.Stack.Endpoints()
}

/*
Cleanup cleans up the underlying Scaffold stack with ctx.
*/
func (r *Stack) Cleanup(ctx context.Context) error {
	return r.Stack.Cleanup(ctx)
}

/*
Logs returns logs from every service in the preset stack.
*/
func (r *Stack) Logs(ctx context.Context) (logs.LogStreams, error) {
	return r.Stack.Logs(ctx)
}

var _ scaffold.Service = (*Stack)(nil)
var _ scaffold.LabelAttachable = (*Stack)(nil)
var _ scaffold.NamePrefixAttachable = (*Stack)(nil)
