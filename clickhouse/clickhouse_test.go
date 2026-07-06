package clickhouse

import (
	"context"
	"testing"

	scaffoldcontainer "github.com/hlfshell/scaffold/container"
)

func TestClickHouseCreateSeedExecCleanup(t *testing.T) {
	if !scaffoldcontainer.DockerAvailable() {
		t.Skip("docker is not available")
	}

	ctx := context.Background()
	service, err := NewClickHouse("scaffold-test-clickhouse", "latest", "default", "secret", "events")
	if err != nil {
		t.Fatal(err)
	}
	service.WithSQL("CREATE TABLE seed_check (id UInt64) ENGINE = Memory")

	if err := service.Create(ctx); err != nil {
		t.Fatal(err)
	}
	defer service.Cleanup(ctx)

	if err := service.Exec(ctx, "INSERT INTO seed_check VALUES (1)"); err != nil {
		t.Fatal(err)
	}
}
