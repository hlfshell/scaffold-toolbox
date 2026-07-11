package litellm

import (
	"context"
	"testing"

	scaffoldcontainer "github.com/hlfshell/scaffold/container"
)

func TestLiteLLMCreateModelsCleanup(t *testing.T) {
	if !scaffoldcontainer.DockerAvailable() {
		t.Skip("docker is not available")
	}

	ctx := context.Background()
	service, err := NewLiteLLM("scaffold-test-litellm", "latest")
	if err != nil {
		t.Fatal(err)
	}

	if err := service.Create(ctx); err != nil {
		t.Fatal(err)
	}
	defer service.Cleanup(ctx)

	if _, err := service.Models(ctx); err != nil {
		t.Fatal(err)
	}
}
