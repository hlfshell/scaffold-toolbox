package memcached

import (
	"context"
	"testing"
	"time"

	scaffoldcontainer "github.com/hlfshell/scaffold/container"
)

func TestMemcachedCreateSeedConnectCleanup(t *testing.T) {
	if !scaffoldcontainer.DockerAvailable() {
		t.Skip("docker is not available")
	}

	ctx := context.Background()
	service, err := NewMemcached("scaffold-test-memcached", "latest")
	if err != nil {
		t.Fatal(err)
	}
	service.WithItem("hello", []byte("world"))

	if err := service.Create(ctx); err != nil {
		t.Fatal(err)
	}
	defer service.Cleanup(ctx)

	client, err := service.ConnectWithTimeout(ctx, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	item, err := client.Get("hello")
	if err != nil {
		t.Fatal(err)
	}
	if string(item.Value) != "world" {
		t.Fatalf("expected seeded value world, got %q", string(item.Value))
	}
}
