package minio

import (
	"context"
	"testing"

	scaffoldcontainer "github.com/hlfshell/scaffold/container"
	minioclient "github.com/minio/minio-go/v7"
)

func TestMinIOCreateSeedClientCleanup(t *testing.T) {
	if !scaffoldcontainer.DockerAvailable() {
		t.Skip("docker is not available")
	}

	ctx := context.Background()
	service, err := NewMinIO("scaffold-test-minio", "latest", "minioadmin", "minioadmin")
	if err != nil {
		t.Fatal(err)
	}
	service.WithObject("uploads", "hello.txt", []byte("hello"), "text/plain")

	if err := service.Create(ctx); err != nil {
		t.Fatal(err)
	}
	defer service.Cleanup(ctx)

	client, err := service.Client()
	if err != nil {
		t.Fatal(err)
	}
	info, err := client.StatObject(ctx, "uploads", "hello.txt", minioclient.StatObjectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if info.Size != 5 {
		t.Fatalf("expected uploaded object size 5, got %d", info.Size)
	}
}
