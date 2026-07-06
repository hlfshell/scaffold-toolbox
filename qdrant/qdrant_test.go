package qdrant

import (
	"context"
	"testing"

	scaffoldcontainer "github.com/hlfshell/scaffold/container"
)

func TestQdrantCreateSeedCleanup(t *testing.T) {
	if !scaffoldcontainer.DockerAvailable() {
		t.Skip("docker is not available")
	}

	ctx := context.Background()
	service, err := NewQdrant("scaffold-test-qdrant", "latest")
	if err != nil {
		t.Fatal(err)
	}
	service.WithCollection(CollectionConfig{Name: "docs", Size: 3, Distance: "Cosine"})
	service.WithPoints("docs", []Point{{ID: 1, Vector: []float64{0.1, 0.2, 0.3}}})

	if err := service.Create(ctx); err != nil {
		t.Fatal(err)
	}
	defer service.Cleanup(ctx)

	if err := service.UpsertPoints(ctx, "docs", []Point{{ID: 2, Vector: []float64{0.4, 0.5, 0.6}}}); err != nil {
		t.Fatal(err)
	}
}
