package qdrant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/hlfshell/scaffold"
	scaffoldcontainer "github.com/hlfshell/scaffold/container"
	"github.com/hlfshell/scaffold/logs"
)

/*
Qdrant is a typed harness around the qdrant/qdrant container. It exposes
REST and gRPC endpoints and provides simple HTTP preload helpers.
*/
type Qdrant struct {
	container *scaffoldcontainer.Container
	client    *http.Client
	name      string
	restPort  string
	grpcPort  string
	preloads  []func(*Qdrant) error
}

/*
CollectionConfig describes the vector collection to create in Qdrant.
*/
type CollectionConfig struct {
	Name     string
	Size     int
	Distance string
}

/*
Point is the minimal JSON shape used by Qdrant's point upsert API.
*/
type Point struct {
	ID      any            `json:"id"`
	Vector  []float64      `json:"vector"`
	Payload map[string]any `json:"payload,omitempty"`
}

/*
NewQdrant creates a Qdrant harness. A blank tag is passed through to
Scaffold and will default to "latest".
*/
func NewQdrant(name string, tag string) (*Qdrant, error) {
	container, err := scaffoldcontainer.NewContainer(
		name,
		"qdrant/qdrant",
		scaffoldcontainer.WithTag(tag),
		scaffoldcontainer.WithPorts(map[string]string{
			"6333": "",
			"6334": "",
		}),
	)
	if err != nil {
		return nil, err
	}

	return &Qdrant{
		container: container,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		name:     name,
		preloads: []func(*Qdrant) error{},
	}, nil
}

/*
Name returns the service name used by Scaffold stacks.
*/
func (q *Qdrant) Name() string {
	return q.name
}

/*
SetNetwork attaches the underlying container to a Docker network when it
is created.
*/
func (q *Qdrant) SetNetwork(name string) {
	q.container.SetNetwork(name)
}

/*
SetLabels merges Docker labels onto the underlying container.
*/
func (q *Qdrant) SetLabels(labels map[string]string) {
	q.container.SetLabels(labels)
}

/*
SetNamePrefix prefixes the underlying Docker container name before it is
created.
*/
func (q *Qdrant) SetNamePrefix(prefix string) {
	q.container.SetNamePrefix(prefix)
}

/*
Create starts Qdrant with ctx, waits for the readiness endpoint,
and runs any registered preload functions.
*/
func (q *Qdrant) Create(ctx context.Context) error {
	err := q.container.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start qdrant container: %w", err)
	}

	ports := q.container.GetPorts()
	q.restPort = ports["6333"]
	q.grpcPort = ports["6334"]

	err = scaffold.WaitForHTTP(ctx, q.Endpoint()+"/readyz", http.StatusOK, 30*time.Second)
	if err != nil {
		q.container.Cleanup(context.WithoutCancel(ctx))
		return fmt.Errorf("qdrant failed to become ready: %w", err)
	}

	err = q.Preload()
	if err != nil {
		q.container.Cleanup(context.WithoutCancel(ctx))
		return err
	}

	return nil
}

/*
Endpoint returns the local Qdrant REST endpoint.
*/
func (q *Qdrant) Endpoint() string {
	return fmt.Sprintf("http://127.0.0.1:%s", q.restPort)
}

/*
GRPCEndpoint returns the local Qdrant gRPC endpoint.
*/
func (q *Qdrant) GRPCEndpoint() string {
	return fmt.Sprintf("127.0.0.1:%s", q.grpcPort)
}

/*
Env returns Qdrant endpoint environment variables.
*/
func (q *Qdrant) Env() map[string]string {
	return map[string]string{
		"QDRANT_URL":      q.Endpoint(),
		"QDRANT_GRPC_URL": q.GRPCEndpoint(),
	}
}

/*
Endpoints returns named Qdrant endpoints.
*/
func (q *Qdrant) Endpoints() map[string]string {
	return map[string]string{
		q.name:           q.Endpoint(),
		q.name + "-grpc": q.GRPCEndpoint(),
	}
}

/*
CreateCollection creates or updates a Qdrant collection using the REST
API.
*/
func (q *Qdrant) CreateCollection(config CollectionConfig) error {
	if config.Distance == "" {
		config.Distance = "Cosine"
	}

	body := map[string]any{
		"vectors": map[string]any{
			"size":     config.Size,
			"distance": config.Distance,
		},
	}

	return q.doJSON(http.MethodPut, fmt.Sprintf("/collections/%s", config.Name), body, nil)
}

/*
UpsertPoints inserts or updates points in a Qdrant collection.
*/
func (q *Qdrant) UpsertPoints(collection string, points []Point) error {
	body := map[string]any{
		"points": points,
	}

	return q.doJSON(http.MethodPut, fmt.Sprintf("/collections/%s/points?wait=true", collection), body, nil)
}

/*
UpsertPointsFromJSON loads points from a JSON file and upserts them into
a collection.
*/
func (q *Qdrant) UpsertPointsFromJSON(collection string, path string) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	points := []Point{}
	err = json.Unmarshal(contents, &points)
	if err != nil {
		return err
	}

	return q.UpsertPoints(collection, points)
}

/*
WithCollection registers a collection to create after Qdrant is ready.
*/
func (q *Qdrant) WithCollection(config CollectionConfig) *Qdrant {
	q.preloads = append(q.preloads, func(qdrant *Qdrant) error {
		return qdrant.CreateCollection(config)
	})

	return q
}

/*
WithPoints registers points to upsert after Qdrant is ready.
*/
func (q *Qdrant) WithPoints(collection string, points []Point) *Qdrant {
	q.preloads = append(q.preloads, func(qdrant *Qdrant) error {
		return qdrant.UpsertPoints(collection, points)
	})

	return q
}

/*
Preload runs all registered Qdrant preload functions.
*/
func (q *Qdrant) Preload() error {
	for _, preload := range q.preloads {
		err := preload(q)
		if err != nil {
			return fmt.Errorf("failed to preload qdrant: %w", err)
		}
	}

	return nil
}

/*
Cleanup removes the Qdrant container.
*/
func (q *Qdrant) Cleanup(ctx context.Context) error {
	return q.container.Cleanup(ctx)
}

/*
Logs returns the Qdrant container logs keyed by service name.
*/
func (q *Qdrant) Logs(ctx context.Context) (logs.LogStreams, error) {
	stream, err := q.container.Logs(ctx)
	if err != nil {
		return nil, err
	}

	return logs.LogStreams{q.name: stream}, nil
}

func (q *Qdrant) doJSON(method string, path string, input any, output any) error {
	payload, err := json.Marshal(input)
	if err != nil {
		return err
	}

	request, err := http.NewRequest(method, q.Endpoint()+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := q.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("qdrant request failed with status %d", response.StatusCode)
	}

	if output != nil {
		return json.NewDecoder(response.Body).Decode(output)
	}

	return nil
}
