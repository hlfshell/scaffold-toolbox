package trino

import (
	"context"
	"strings"
	"testing"

	scaffoldcontainer "github.com/hlfshell/scaffold/container"
)

func TestTrinoCreateQueryCleanup(t *testing.T) {
	if !scaffoldcontainer.DockerAvailable() {
		t.Skip("docker is not available")
	}

	ctx := context.Background()
	service, err := NewTrino("scaffold-test-trino", "latest", WithMemoryCatalog("memory"))
	if err != nil {
		t.Fatal(err)
	}

	if err := service.Create(ctx); err != nil {
		t.Fatal(err)
	}
	defer service.Cleanup(ctx)

	body, err := service.Query(ctx, "select 1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "nextUri") && !strings.Contains(string(body), "data") {
		t.Fatalf("expected trino query response, got %s", string(body))
	}
}
