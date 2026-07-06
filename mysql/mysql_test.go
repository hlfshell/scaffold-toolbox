package mysql

import (
	"context"
	"testing"
	"time"

	scaffoldcontainer "github.com/hlfshell/scaffold/container"
)

func TestMysqlCreateSeedConnectCleanup(t *testing.T) {
	if !scaffoldcontainer.DockerAvailable() {
		t.Skip("docker is not available")
	}

	ctx := context.Background()
	service, err := NewMysql("scaffold-test-mysql", "8", "user", "pass", "test")
	if err != nil {
		t.Fatal(err)
	}
	service.WithSQL("create table seed_check (id int primary key)")

	if err := service.Create(ctx); err != nil {
		t.Fatal(err)
	}
	defer service.Cleanup(ctx)

	db, err := service.ConnectWithTimeout(ctx, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
}
