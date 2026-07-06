package weaviate

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
Weaviate is a typed harness around the semitechnologies/weaviate image.
It exposes the REST endpoint and simple schema/object preload helpers.
*/
type Weaviate struct {
	container *scaffoldcontainer.Container
	client    *http.Client
	name      string
	port      string
	preloads  []func(context.Context, *Weaviate) error
}

/*
Class is the minimal schema class shape accepted by Weaviate's REST API.
Callers can use Raw for provider-specific schema fields.
*/
type Class struct {
	Class      string         `json:"class"`
	Vectorizer string         `json:"vectorizer,omitempty"`
	Properties []Property     `json:"properties,omitempty"`
	Raw        map[string]any `json:"-"`
}

/*
Property describes a Weaviate class property.
*/
type Property struct {
	Name     string   `json:"name"`
	DataType []string `json:"dataType"`
}

/*
Object describes a Weaviate object to create during preload.
*/
type Object struct {
	Class      string         `json:"class"`
	ID         string         `json:"id,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
	Vector     []float64      `json:"vector,omitempty"`
}

/*
NewWeaviate creates a Weaviate service with anonymous access and no
default vectorizer, which is the most predictable local-test default.
*/
func NewWeaviate(name string, tag string) (*Weaviate, error) {
	container, err := scaffoldcontainer.NewContainer(
		name,
		"semitechnologies/weaviate",
		scaffoldcontainer.WithTag(tag),
		scaffoldcontainer.WithPort("8080", ""),
		scaffoldcontainer.WithEnv(map[string]string{
			"AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED": "true",
			"DEFAULT_VECTORIZER_MODULE":               "none",
			"ENABLE_MODULES":                          "",
			"PERSISTENCE_DATA_PATH":                   "/var/lib/weaviate",
			"QUERY_DEFAULTS_LIMIT":                    "25",
		}),
	)
	if err != nil {
		return nil, err
	}

	return &Weaviate{
		container: container,
		client:    &http.Client{Timeout: 15 * time.Second},
		name:      name,
		preloads:  []func(context.Context, *Weaviate) error{},
	}, nil
}

func (w *Weaviate) Name() string {
	return w.name
}

func (w *Weaviate) SetNetwork(name string) {
	w.container.SetNetwork(name)
}

func (w *Weaviate) SetLabels(labels map[string]string) {
	w.container.SetLabels(labels)
}

func (w *Weaviate) SetNamePrefix(prefix string) {
	w.container.SetNamePrefix(prefix)
}

/*
Create starts Weaviate, waits for readiness, and runs schema/object
preload functions.
*/
func (w *Weaviate) Create(ctx context.Context) error {
	err := w.container.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start weaviate container: %w", err)
	}

	ports := w.container.GetPorts()
	w.port = ports["8080"]

	err = scaffold.WaitForHTTP(ctx, w.Endpoint()+"/v1/.well-known/ready", http.StatusOK, 60*time.Second)
	if err != nil {
		w.container.Cleanup(context.WithoutCancel(ctx))
		return fmt.Errorf("weaviate failed to become ready: %w", err)
	}

	err = w.Preload(ctx)
	if err != nil {
		w.container.Cleanup(context.WithoutCancel(ctx))
		return err
	}

	return nil
}

/*
Endpoint returns the local Weaviate REST endpoint.
*/
func (w *Weaviate) Endpoint() string {
	return fmt.Sprintf("http://127.0.0.1:%s", w.port)
}

func (w *Weaviate) Env() map[string]string {
	return map[string]string{
		"WEAVIATE_URL": w.Endpoint(),
	}
}

func (w *Weaviate) Endpoints() map[string]string {
	return map[string]string{
		w.name: w.Endpoint(),
	}
}

/*
CreateClass creates a schema class. Raw fields are merged into the JSON
body before the request is sent.
*/
func (w *Weaviate) CreateClass(ctx context.Context, class Class) error {
	body := map[string]any{}
	for key, value := range class.Raw {
		body[key] = value
	}
	body["class"] = class.Class
	if class.Vectorizer != "" {
		body["vectorizer"] = class.Vectorizer
	}
	if len(class.Properties) > 0 {
		body["properties"] = class.Properties
	}

	return w.doJSON(ctx, http.MethodPost, "/v1/schema", body, nil)
}

/*
CreateObject creates one object through the REST API.
*/
func (w *Weaviate) CreateObject(ctx context.Context, object Object) error {
	return w.doJSON(ctx, http.MethodPost, "/v1/objects", object, nil)
}

/*
WithClass registers a schema class to create after Weaviate is ready.
*/
func (w *Weaviate) WithClass(class Class) *Weaviate {
	w.preloads = append(w.preloads, func(ctx context.Context, weaviate *Weaviate) error {
		return weaviate.CreateClass(ctx, class)
	})

	return w
}

/*
WithObject registers an object to create after Weaviate is ready.
*/
func (w *Weaviate) WithObject(object Object) *Weaviate {
	w.preloads = append(w.preloads, func(ctx context.Context, weaviate *Weaviate) error {
		return weaviate.CreateObject(ctx, object)
	})

	return w
}

/*
Preload runs all registered Weaviate setup functions.
*/
func (w *Weaviate) Preload(ctx context.Context) error {
	for _, preload := range w.preloads {
		err := preload(ctx, w)
		if err != nil {
			return fmt.Errorf("failed to preload weaviate: %w", err)
		}
	}

	return nil
}

func (w *Weaviate) Cleanup(ctx context.Context) error {
	return w.container.Cleanup(ctx)
}

func (w *Weaviate) Logs(ctx context.Context) (logs.LogStreams, error) {
	stream, err := w.container.Logs(ctx)
	if err != nil {
		return nil, err
	}

	return logs.LogStreams{w.name: stream}, nil
}

func (w *Weaviate) doJSON(ctx context.Context, method string, path string, input any, output any) error {
	payload, err := json.Marshal(input)
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, method, w.Endpoint()+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := w.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("weaviate request failed with status %d", response.StatusCode)
	}

	if output != nil {
		return json.NewDecoder(response.Body).Decode(output)
	}

	return nil
}
