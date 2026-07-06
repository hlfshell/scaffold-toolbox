package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hlfshell/scaffold"
	scaffoldcontainer "github.com/hlfshell/scaffold/container"
	"github.com/hlfshell/scaffold/logs"
)

/*
Ollama is a typed harness around the ollama/ollama container. It exposes
the local API endpoint and model preload helpers.
*/
type Ollama struct {
	container *scaffoldcontainer.Container
	client    *http.Client
	name      string
	port      string
	preloads  []func(context.Context, *Ollama) error
}

/*
NewOllama creates an Ollama service using the official image.
*/
func NewOllama(name string, tag string) (*Ollama, error) {
	container, err := scaffoldcontainer.NewContainer(
		name,
		"ollama/ollama",
		scaffoldcontainer.WithTag(tag),
		scaffoldcontainer.WithPort("11434", ""),
	)
	if err != nil {
		return nil, err
	}

	return &Ollama{
		container: container,
		client:    &http.Client{Timeout: 30 * time.Second},
		name:      name,
		preloads:  []func(context.Context, *Ollama) error{},
	}, nil
}

func (o *Ollama) Name() string {
	return o.name
}

func (o *Ollama) SetNetwork(name string) {
	o.container.SetNetwork(name)
}

func (o *Ollama) SetLabels(labels map[string]string) {
	o.container.SetLabels(labels)
}

func (o *Ollama) SetNamePrefix(prefix string) {
	o.container.SetNamePrefix(prefix)
}

/*
Create starts Ollama, waits for the API to respond, and pulls any
registered models.
*/
func (o *Ollama) Create(ctx context.Context) error {
	err := o.container.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start ollama container: %w", err)
	}

	ports := o.container.GetPorts()
	o.port = ports["11434"]

	err = scaffold.WaitForHTTP(ctx, o.Endpoint()+"/api/tags", http.StatusOK, 45*time.Second)
	if err != nil {
		o.container.Cleanup(context.WithoutCancel(ctx))
		return fmt.Errorf("ollama failed to become ready: %w", err)
	}

	err = o.Preload(ctx)
	if err != nil {
		o.container.Cleanup(context.WithoutCancel(ctx))
		return err
	}

	return nil
}

/*
Endpoint returns the local Ollama API endpoint.
*/
func (o *Ollama) Endpoint() string {
	return fmt.Sprintf("http://127.0.0.1:%s", o.port)
}

func (o *Ollama) Env() map[string]string {
	return map[string]string{
		"OLLAMA_HOST": o.Endpoint(),
	}
}

func (o *Ollama) Endpoints() map[string]string {
	return map[string]string{
		o.name: o.Endpoint(),
	}
}

/*
PullModel downloads a model into the running Ollama service.
*/
func (o *Ollama) PullModel(ctx context.Context, model string) error {
	body := map[string]any{
		"name":   model,
		"stream": false,
	}

	return o.doJSON(ctx, "/api/pull", body, nil)
}

/*
WithModel registers a model to pull after Ollama is ready.
*/
func (o *Ollama) WithModel(model string) *Ollama {
	o.preloads = append(o.preloads, func(ctx context.Context, ollama *Ollama) error {
		return ollama.PullModel(ctx, model)
	})

	return o
}

/*
Preload runs all registered Ollama model setup functions.
*/
func (o *Ollama) Preload(ctx context.Context) error {
	for _, preload := range o.preloads {
		err := preload(ctx, o)
		if err != nil {
			return fmt.Errorf("failed to preload ollama: %w", err)
		}
	}

	return nil
}

func (o *Ollama) Cleanup(ctx context.Context) error {
	return o.container.Cleanup(ctx)
}

func (o *Ollama) Logs(ctx context.Context) (logs.LogStreams, error) {
	stream, err := o.container.Logs(ctx)
	if err != nil {
		return nil, err
	}

	return logs.LogStreams{o.name: stream}, nil
}

func (o *Ollama) doJSON(ctx context.Context, path string, input any, output any) error {
	payload, err := json.Marshal(input)
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, o.Endpoint()+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := o.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("ollama request failed with status %d", response.StatusCode)
	}

	if output != nil {
		return json.NewDecoder(response.Body).Decode(output)
	}

	return nil
}
