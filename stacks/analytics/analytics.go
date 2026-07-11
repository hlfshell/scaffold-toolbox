package analytics

import (
	"context"

	"github.com/hlfshell/scaffold"
	clickhouse "github.com/hlfshell/scaffold-toolbox/clickhouse"
	trino "github.com/hlfshell/scaffold-toolbox/trino"
	"github.com/hlfshell/scaffold/logs"
)

/*
Stack is a local analytics workspace. It combines ClickHouse for
fast analytical storage with Trino for SQL federation experiments.
*/
type Stack struct {
	Stack      *scaffold.Stack
	name       string
	ClickHouse *clickhouse.ClickHouse
	Trino      *trino.Trino
}

// Option configures the analytics stack before it is built.
type Option func(*analyticsConfig)

type analyticsConfig struct {
	clickhouseTag      string
	clickhouseUser     string
	clickhousePassword string
	clickhouseDatabase string
	clickhouseSQL      []string
	clickhouseSQLFiles []string
	trinoTag           string
	trinoOptions       []trino.Option
}

/*
WithClickHouseSQL registers SQL to run after ClickHouse is ready.
*/
func WithClickHouseSQL(sql string) Option {
	return func(config *analyticsConfig) {
		config.clickhouseSQL = append(config.clickhouseSQL, sql)
	}
}

/*
WithClickHouseSQLFile registers a SQL file to run after ClickHouse is
ready.
*/
func WithClickHouseSQLFile(path string) Option {
	return func(config *analyticsConfig) {
		config.clickhouseSQLFiles = append(config.clickhouseSQLFiles, path)
	}
}

/*
WithTrinoCatalog adds a Trino catalog to the analytics stack.
*/
func WithTrinoCatalog(name string, properties map[string]string) Option {
	return func(config *analyticsConfig) {
		config.trinoOptions = append(config.trinoOptions, trino.WithCatalog(name, properties))
	}
}

/*
WithTrinoMemoryCatalog adds a memory catalog to Trino.
*/
func WithTrinoMemoryCatalog(name string) Option {
	return func(config *analyticsConfig) {
		config.trinoOptions = append(config.trinoOptions, trino.WithMemoryCatalog(name))
	}
}

/*
WithImageTags sets the ClickHouse and Trino image tags.
*/
func WithImageTags(clickhouseTag string, trinoTag string) Option {
	return func(config *analyticsConfig) {
		if clickhouseTag != "" {
			config.clickhouseTag = clickhouseTag
		}
		if trinoTag != "" {
			config.trinoTag = trinoTag
		}
	}
}

/*
NewStack creates a ClickHouse and Trino stack on a shared Docker
network.
*/
func NewStack(name string, options ...Option) (*Stack, error) {
	config := &analyticsConfig{
		clickhouseTag:      "latest",
		clickhouseUser:     "scaffold",
		clickhousePassword: "scaffold",
		clickhouseDatabase: "analytics",
		trinoTag:           "latest",
	}
	for _, option := range options {
		option(config)
	}

	store, err := clickhouse.NewClickHouse(name+"-clickhouse", config.clickhouseTag, config.clickhouseUser, config.clickhousePassword, config.clickhouseDatabase)
	if err != nil {
		return nil, err
	}
	for _, sql := range config.clickhouseSQL {
		store.WithSQL(sql)
	}
	for _, path := range config.clickhouseSQLFiles {
		store.WithSQLFile(path)
	}

	query, err := trino.NewTrino(name+"-trino", config.trinoTag, config.trinoOptions...)
	if err != nil {
		return nil, err
	}

	stack := &Stack{
		name:       name,
		ClickHouse: store,
		Trino:      query,
	}
	stack.Stack = scaffold.NewStack(
		name,
		scaffold.WithServices(stack.ClickHouse, stack.Trino),
		scaffold.WithSharedNetwork(),
	)

	return stack, nil
}

func (a *Stack) Name() string {
	return a.name
}

/*
SetLabels passes inherited labels to the underlying scaffold stack.
*/
func (a *Stack) SetLabels(labels map[string]string) {
	a.Stack.SetLabels(labels)
}

/*
SetNamePrefix passes an inherited Docker name prefix to the underlying
scaffold stack.
*/
func (a *Stack) SetNamePrefix(prefix string) {
	a.Stack.SetNamePrefix(prefix)
}

/*
Create starts ClickHouse and Trino.
*/
func (a *Stack) Create(ctx context.Context) error {
	return a.Stack.Create(ctx)
}

/*
IsRunning reports whether any labeled resources for this stack are
running.
*/
func (a *Stack) IsRunning(ctx context.Context) (bool, error) {
	return a.Stack.IsRunning(ctx)
}

/*
Resources returns Docker resources discovered for this stack.
*/
func (a *Stack) Resources(ctx context.Context) (scaffold.ResourceStatus, error) {
	return a.Stack.Resources(ctx)
}

/*
Env returns environment variables exposed by ClickHouse and Trino.
*/
func (a *Stack) Env() map[string]string {
	return a.Stack.Env()
}

/*
Endpoints returns ClickHouse and Trino endpoints.
*/
func (a *Stack) Endpoints() map[string]string {
	return a.Stack.Endpoints()
}

/*
ExecClickHouse runs SQL against ClickHouse.
*/
func (a *Stack) ExecClickHouse(ctx context.Context, sql string) error {
	return a.ClickHouse.Exec(ctx, sql)
}

/*
QueryTrino runs SQL through Trino and returns the first response body.
*/
func (a *Stack) QueryTrino(ctx context.Context, sql string) ([]byte, error) {
	return a.Trino.Query(ctx, sql)
}

/*
Cleanup removes resources created by the analytics stack.
*/
func (a *Stack) Cleanup(ctx context.Context) error {
	return a.Stack.Cleanup(ctx)
}

/*
Logs returns logs from the analytics stack services.
*/
func (a *Stack) Logs(ctx context.Context) (logs.LogStreams, error) {
	return a.Stack.Logs(ctx)
}

var _ scaffold.Service = (*Stack)(nil)
var _ scaffold.LabelAttachable = (*Stack)(nil)
var _ scaffold.NamePrefixAttachable = (*Stack)(nil)
