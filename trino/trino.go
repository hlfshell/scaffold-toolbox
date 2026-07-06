package trino

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hlfshell/scaffold"
	scaffoldcontainer "github.com/hlfshell/scaffold/container"
	"github.com/hlfshell/scaffold/logs"
)

/*
Trino is a typed harness around the official Trino container. It runs a
single local coordinator and mounts generated catalog property files.
*/
type Trino struct {
	container  *scaffoldcontainer.Container
	client     *http.Client
	name       string
	port       string
	user       string
	catalogs   map[string]map[string]string
	catalogDir string
}

// Option configures Trino before the container is created.
type Option func(*Trino)

/*
WithUser sets the Trino user sent with HTTP SQL requests.
*/
func WithUser(user string) Option {
	return func(trino *Trino) {
		if user != "" {
			trino.user = user
		}
	}
}

/*
WithCatalog adds a Trino catalog properties file. The map is written as
key=value lines to /etc/trino/catalog/<name>.properties.
*/
func WithCatalog(name string, properties map[string]string) Option {
	return func(trino *Trino) {
		trino.catalogs[name] = clone(properties)
	}
}

/*
WithMemoryCatalog adds Trino's built-in memory connector.
*/
func WithMemoryCatalog(name string) Option {
	return WithCatalog(name, map[string]string{
		"connector.name": "memory",
	})
}

/*
NewTrino creates a Trino service. If no catalogs are supplied, a memory
catalog named "memory" is created so the service is useful immediately.
*/
func NewTrino(name string, tag string, options ...Option) (*Trino, error) {
	service := &Trino{
		name:     name,
		user:     "scaffold",
		client:   &http.Client{Timeout: 30 * time.Second},
		catalogs: map[string]map[string]string{},
	}
	for _, option := range options {
		option(service)
	}
	if len(service.catalogs) == 0 {
		WithMemoryCatalog("memory")(service)
	}

	catalogDir, err := os.MkdirTemp("", "scaffold-trino-catalog-*")
	if err != nil {
		return nil, err
	}
	service.catalogDir = catalogDir

	err = service.writeCatalogs()
	if err != nil {
		os.RemoveAll(catalogDir)
		return nil, err
	}

	container, err := scaffoldcontainer.NewContainer(
		name,
		"trinodb/trino",
		scaffoldcontainer.WithTag(tag),
		scaffoldcontainer.WithPort("8080", ""),
		scaffoldcontainer.WithBind(catalogDir, "/etc/trino/catalog"),
	)
	if err != nil {
		os.RemoveAll(catalogDir)
		return nil, err
	}
	service.container = container

	return service, nil
}

func (t *Trino) Name() string {
	return t.name
}

func (t *Trino) SetNetwork(name string) {
	t.container.SetNetwork(name)
}

func (t *Trino) SetLabels(labels map[string]string) {
	t.container.SetLabels(labels)
}

func (t *Trino) SetNamePrefix(prefix string) {
	t.container.SetNamePrefix(prefix)
}

/*
Create starts Trino and waits for the coordinator info endpoint.
*/
func (t *Trino) Create(ctx context.Context) error {
	err := t.container.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start trino container: %w", err)
	}

	ports := t.container.GetPorts()
	t.port = ports["8080"]

	err = scaffold.WaitForHTTP(ctx, t.Endpoint()+"/v1/info", http.StatusOK, 90*time.Second)
	if err != nil {
		t.container.Cleanup(context.WithoutCancel(ctx))
		return fmt.Errorf("trino failed to become ready: %w", err)
	}

	return nil
}

/*
Endpoint returns the local Trino coordinator endpoint.
*/
func (t *Trino) Endpoint() string {
	return fmt.Sprintf("http://127.0.0.1:%s", t.port)
}

func (t *Trino) Env() map[string]string {
	return map[string]string{
		"TRINO_URL":  t.Endpoint(),
		"TRINO_USER": t.user,
	}
}

func (t *Trino) Endpoints() map[string]string {
	return map[string]string{
		t.name: t.Endpoint(),
	}
}

/*
Query submits SQL to Trino and returns the first HTTP response body.
Callers that need full pagination can follow nextUri from the returned
JSON payload with their own client.
*/
func (t *Trino) Query(ctx context.Context, sql string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, t.Endpoint()+"/v1/statement", bytes.NewBufferString(sql))
	if err != nil {
		return nil, err
	}
	request.Header.Set("X-Trino-User", t.user)

	response, err := t.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := ioReadAll(response)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("trino query failed with status %d: %s", response.StatusCode, string(body))
	}

	return body, nil
}

/*
Cleanup removes the Trino container and generated catalog directory.
*/
func (t *Trino) Cleanup(ctx context.Context) error {
	err := t.container.Cleanup(ctx)
	if t.catalogDir != "" {
		if removeErr := os.RemoveAll(t.catalogDir); err == nil {
			err = removeErr
		}
	}

	return err
}

func (t *Trino) Logs(ctx context.Context) (logs.LogStreams, error) {
	stream, err := t.container.Logs(ctx)
	if err != nil {
		return nil, err
	}

	return logs.LogStreams{t.name: stream}, nil
}

func (t *Trino) writeCatalogs() error {
	for name, properties := range t.catalogs {
		path := filepath.Join(t.catalogDir, name+".properties")
		lines := make([]string, 0, len(properties))
		for key, value := range properties {
			lines = append(lines, key+"="+value)
		}
		sort.Strings(lines)

		err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600)
		if err != nil {
			return err
		}
	}

	return nil
}

func clone(input map[string]string) map[string]string {
	output := map[string]string{}
	for key, value := range input {
		output[key] = value
	}

	return output
}

func ioReadAll(response *http.Response) ([]byte, error) {
	buffer := bytes.Buffer{}
	_, err := buffer.ReadFrom(response.Body)
	return buffer.Bytes(), err
}
