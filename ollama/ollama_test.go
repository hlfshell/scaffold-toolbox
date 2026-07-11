package ollama

import (
	"context"
	"testing"

	scaffoldcontainer "github.com/hlfshell/scaffold/container"
)

func TestOllamaCreateCleanup(t *testing.T) {
	if !scaffoldcontainer.DockerAvailable() {
		t.Skip("docker is not available")
	}

	ctx := context.Background()
	service, err := NewOllama("scaffold-test-ollama", "latest")
	if err != nil {
		t.Fatal(err)
	}

	if err := service.Create(ctx); err != nil {
		t.Fatal(err)
	}
	defer service.Cleanup(ctx)

	if service.Endpoint() == "" {
		t.Fatal("expected ollama endpoint")
	}
}
