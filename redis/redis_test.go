package redis

import (
	"context"
	"testing"
	"time"

	scaffoldcontainer "github.com/hlfshell/scaffold/container"
)

func TestRedisCreateSeedConnectCleanup(t *testing.T) {
	if !scaffoldcontainer.DockerAvailable() {
		t.Skip("docker is not available")
	}

	ctx := context.Background()
	service, err := NewRedis("scaffold-test-redis", "latest")
	if err != nil {
		t.Fatal(err)
	}
	service.WithKey("feature:test", "enabled")

	if err := service.Create(ctx); err != nil {
		t.Fatal(err)
	}
	defer service.Cleanup(ctx)

	client, err := service.ConnectWithTimeout(ctx, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	value, err := client.Get(ctx, "feature:test").Result()
	if err != nil {
		t.Fatal(err)
	}
	if value != "enabled" {
		t.Fatalf("expected seeded value enabled, got %q", value)
	}
}
