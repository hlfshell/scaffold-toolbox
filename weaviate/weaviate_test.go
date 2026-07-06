package weaviate

import (
	"context"
	"testing"

	scaffoldcontainer "github.com/hlfshell/scaffold/container"
)

func TestWeaviateCreateSeedCleanup(t *testing.T) {
	if !scaffoldcontainer.DockerAvailable() {
		t.Skip("docker is not available")
	}

	ctx := context.Background()
	service, err := NewWeaviate("scaffold-test-weaviate", "latest")
	if err != nil {
		t.Fatal(err)
	}
	service.WithClass(Class{
		Class:      "Document",
		Vectorizer: "none",
		Properties: []Property{
			{Name: "body", DataType: []string{"text"}},
		},
	})

	if err := service.Create(ctx); err != nil {
		t.Fatal(err)
	}
	defer service.Cleanup(ctx)

	if err := service.CreateObject(ctx, Object{
		Class:      "Document",
		Properties: map[string]any{"body": "hello"},
		Vector:     []float64{0.1, 0.2, 0.3},
	}); err != nil {
		t.Fatal(err)
	}
}
