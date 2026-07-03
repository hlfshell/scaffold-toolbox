package presets

import (
	"context"

	"github.com/hlfshell/scaffold"
	minio "github.com/hlfshell/scaffold-toolbox/minio"
	postgres "github.com/hlfshell/scaffold-toolbox/postgres"
	qdrant "github.com/hlfshell/scaffold-toolbox/qdrant"
	"github.com/hlfshell/scaffold/logs"
)

/*
RAGStack is a preset stack for local retrieval-augmented generation
development. It composes Postgres, Qdrant, and MinIO.
*/
type RAGStack struct {
	Stack    *scaffold.Stack
	name     string
	Postgres *postgres.Postgres
	Qdrant   *qdrant.Qdrant
	MinIO    *minio.MinIO
}

// RAGOption configures the RAG stack before the underlying Stack is built.
type RAGOption func(*RAGStack)

/*
NewRAGStack creates a local RAG development stack using the toolbox
harnesses. The preset does not duplicate service implementation.
*/
func NewRAGStack(name string, options ...RAGOption) (*RAGStack, error) {
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

	rag := &RAGStack{
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
Name returns the preset stack name.
*/
func (r *RAGStack) Name() string {
	return r.name
}

/*
SetLabels passes inherited labels to the underlying stack.
*/
func (r *RAGStack) SetLabels(labels map[string]string) {
	r.Stack.SetLabels(labels)
}

/*
SetNamePrefix passes an inherited name prefix to the underlying stack.
*/
func (r *RAGStack) SetNamePrefix(prefix string) {
	r.Stack.SetNamePrefix(prefix)
}

/*
WithPostgresSQL registers SQL to preload into the Postgres service.
*/
func WithPostgresSQL(sql string) RAGOption {
	return func(r *RAGStack) {
		r.Postgres.WithSQL(sql)
	}
}

/*
WithQdrantCollection registers a Qdrant collection to create during
preload.
*/
func WithQdrantCollection(config qdrant.CollectionConfig) RAGOption {
	return func(r *RAGStack) {
		r.Qdrant.WithCollection(config)
	}
}

/*
WithMinIOBucket registers a MinIO bucket to create during preload.
*/
func WithMinIOBucket(bucket string) RAGOption {
	return func(r *RAGStack) {
		r.MinIO.WithBucket(bucket)
	}
}

/*
Create creates the services in the underlying Scaffold stack with ctx.
*/
func (r *RAGStack) Create(ctx context.Context) error {
	return r.Stack.Create(ctx)
}

/*
IsRunning returns true if any matching container for this preset stack is
running.
*/
func (r *RAGStack) IsRunning(ctx context.Context) (bool, error) {
	return r.Stack.IsRunning(ctx)
}

/*
Resources returns matching Docker resources for this preset stack.
*/
func (r *RAGStack) Resources(ctx context.Context) (scaffold.ResourceStatus, error) {
	return r.Stack.Resources(ctx)
}

/*
Env returns environment variables from the underlying stack.
*/
func (r *RAGStack) Env() map[string]string {
	return r.Stack.Env()
}

/*
Endpoints returns endpoints from the underlying stack.
*/
func (r *RAGStack) Endpoints() map[string]string {
	return r.Stack.Endpoints()
}

/*
Cleanup cleans up the underlying Scaffold stack with ctx.
*/
func (r *RAGStack) Cleanup(ctx context.Context) error {
	return r.Stack.Cleanup(ctx)
}

/*
Logs returns logs from every service in the preset stack.
*/
func (r *RAGStack) Logs(ctx context.Context) (logs.LogStreams, error) {
	return r.Stack.Logs(ctx)
}

var _ scaffold.Service = (*RAGStack)(nil)
var _ scaffold.LabelAttachable = (*RAGStack)(nil)
var _ scaffold.NamePrefixAttachable = (*RAGStack)(nil)
