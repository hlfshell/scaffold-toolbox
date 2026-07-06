package postgres

import (
	"context"
	"testing"
	"time"

	scaffoldcontainer "github.com/hlfshell/scaffold/container"
)

func TestPostgresCreateConnectCleanup(t *testing.T) {
	if !scaffoldcontainer.DockerAvailable() {
		t.Skip("docker is not available")
	}

	postgres, err := NewPostgres("scaffold-test-postgres", "latest", "user", "pass", "test")
	if err != nil {
		t.Fatal(err)
	}
	postgres.WithSQL("create table if not exists seed_check (id int primary key);")

	ctx := context.Background()

	err = postgres.Create(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer postgres.Cleanup(ctx)

	db, err := postgres.ConnectWithTimeout(ctx, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		t.Fatal(err)
	}
}
