package mongo

import (
	"context"
	"testing"

	scaffoldcontainer "github.com/hlfshell/scaffold/container"
)

func TestMongoCreateSeedConnectCleanup(t *testing.T) {
	if !scaffoldcontainer.DockerAvailable() {
		t.Skip("docker is not available")
	}

	ctx := context.Background()
	service, err := NewMongo("scaffold-test-mongo", "latest", "root", "secret", "app")
	if err != nil {
		t.Fatal(err)
	}
	service.WithDocuments("users", map[string]any{"name": "Ada"})

	if err := service.Create(ctx); err != nil {
		t.Fatal(err)
	}
	defer service.Cleanup(ctx)

	client := service.Client()
	if client == nil {
		t.Fatal("expected mongo client after create")
	}

	count, err := client.Database("app").Collection("users").CountDocuments(ctx, map[string]any{"name": "Ada"})
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one seeded document, got %d", count)
	}
}
