package litellm

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
LiteLLM is a typed harness around the LiteLLM proxy container. It exposes
an OpenAI-compatible endpoint for local provider/proxy testing.
*/
type LiteLLM struct {
	container *scaffoldcontainer.Container
	client    *http.Client
	name      string
	masterKey string
	port      string
}

// Option configures the LiteLLM container before it is created.
type Option func(*config)

type config struct {
	masterKey  string
	configFile string
	env        map[string]string
}

/*
WithMasterKey sets the bearer token expected by the proxy.
*/
func WithMasterKey(key string) Option {
	return func(config *config) {
		if key != "" {
			config.masterKey = key
		}
	}
}

/*
WithConfigFile mounts a LiteLLM proxy config file into the container and
starts the proxy with that config.
*/
func WithConfigFile(path string) Option {
	return func(config *config) {
		config.configFile = path
	}
}

/*
WithEnv adds provider keys or other LiteLLM environment values to the
container.
*/
func WithEnv(env map[string]string) Option {
	return func(config *config) {
		for key, value := range env {
			config.env[key] = value
		}
	}
}

/*
NewLiteLLM creates a LiteLLM proxy service. Without options it starts an
empty proxy with a local master key, which is useful for checking that
applications can reach an OpenAI-compatible API surface.
*/
func NewLiteLLM(name string, tag string, options ...Option) (*LiteLLM, error) {
	config := &config{
		masterKey: "sk-local",
		env:       map[string]string{},
	}
	for _, option := range options {
		option(config)
	}

	env := map[string]string{
		"LITELLM_MASTER_KEY": config.masterKey,
	}
	for key, value := range config.env {
		env[key] = value
	}

	containerOptions := []scaffoldcontainer.ContainerOption{
		scaffoldcontainer.WithTag(tag),
		scaffoldcontainer.WithPort("4000", ""),
		scaffoldcontainer.WithEnv(env),
		scaffoldcontainer.WithCommand("--host", "0.0.0.0", "--port", "4000"),
	}
	if config.configFile != "" {
		containerOptions = append(
			containerOptions,
			scaffoldcontainer.WithBind(config.configFile, "/app/config.yaml"),
			scaffoldcontainer.WithCommand("--config", "/app/config.yaml", "--host", "0.0.0.0", "--port", "4000"),
		)
	}

	container, err := scaffoldcontainer.NewContainer(
		name,
		"ghcr.io/berriai/litellm",
		containerOptions...,
	)
	if err != nil {
		return nil, err
	}

	return &LiteLLM{
		container: container,
		client:    &http.Client{Timeout: 30 * time.Second},
		name:      name,
		masterKey: config.masterKey,
	}, nil
}

func (l *LiteLLM) Name() string {
	return l.name
}

func (l *LiteLLM) SetNetwork(name string) {
	l.container.SetNetwork(name)
}

func (l *LiteLLM) SetLabels(labels map[string]string) {
	l.container.SetLabels(labels)
}

func (l *LiteLLM) SetNamePrefix(prefix string) {
	l.container.SetNamePrefix(prefix)
}

/*
Create starts LiteLLM and waits for the readiness endpoint.
*/
func (l *LiteLLM) Create(ctx context.Context) error {
	err := l.container.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start litellm container: %w", err)
	}

	ports := l.container.GetPorts()
	l.port = ports["4000"]

	err = scaffold.WaitForHTTP(ctx, l.Endpoint()+"/health/readiness", http.StatusOK, 60*time.Second)
	if err != nil {
		l.container.Cleanup(context.WithoutCancel(ctx))
		return fmt.Errorf("litellm failed to become ready: %w", err)
	}

	return nil
}

/*
Endpoint returns the local OpenAI-compatible proxy endpoint.
*/
func (l *LiteLLM) Endpoint() string {
	return fmt.Sprintf("http://127.0.0.1:%s", l.port)
}

func (l *LiteLLM) Env() map[string]string {
	return map[string]string{
		"LITELLM_URL":     l.Endpoint(),
		"OPENAI_BASE_URL": l.Endpoint(),
		"OPENAI_API_KEY":  l.masterKey,
	}
}

func (l *LiteLLM) Endpoints() map[string]string {
	return map[string]string{
		l.name: l.Endpoint(),
	}
}

/*
Models returns the proxy's OpenAI-compatible model listing.
*/
func (l *LiteLLM) Models(ctx context.Context) (map[string]any, error) {
	output := map[string]any{}
	err := l.doJSON(ctx, http.MethodGet, "/v1/models", nil, &output)
	return output, err
}

func (l *LiteLLM) Cleanup(ctx context.Context) error {
	return l.container.Cleanup(ctx)
}

func (l *LiteLLM) Logs(ctx context.Context) (logs.LogStreams, error) {
	stream, err := l.container.Logs(ctx)
	if err != nil {
		return nil, err
	}

	return logs.LogStreams{l.name: stream}, nil
}

func (l *LiteLLM) doJSON(ctx context.Context, method string, path string, input any, output any) error {
	var body *bytes.Reader
	if input == nil {
		body = bytes.NewReader(nil)
	} else {
		payload, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(payload)
	}

	request, err := http.NewRequestWithContext(ctx, method, l.Endpoint()+path, body)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	if l.masterKey != "" {
		request.Header.Set("Authorization", "Bearer "+l.masterKey)
	}

	response, err := l.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("litellm request failed with status %d", response.StatusCode)
	}

	if output != nil {
		return json.NewDecoder(response.Body).Decode(output)
	}

	return nil
}
