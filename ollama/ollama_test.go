package ollama

import (
	"context"
	"os"
	"testing"

	scaffoldcontainer "github.com/hlfshell/scaffold/container"
)

func TestOllamaCreateCleanup(t *testing.T) {
	if os.Getenv("SCAFFOLD_TOOLBOX_LLM_TESTS") != "1" {
		t.Skip("set SCAFFOLD_TOOLBOX_LLM_TESTS=1 to run LLM integration tests")
	}
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
