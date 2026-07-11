package analytics

import "testing"

func TestNewStackBuildsComposedServices(t *testing.T) {
	stack, err := NewStack(
		"scaffold-test-analytics",
		WithClickHouseSQL("create table events (id UInt64) engine = Memory;"),
		WithTrinoMemoryCatalog("scratch"),
	)
	if err != nil {
		t.Fatal(err)
	}

	if stack.ClickHouse == nil {
		t.Fatal("expected clickhouse service")
	}
	if stack.Trino == nil {
		t.Fatal("expected trino service")
	}
	if stack.Stack == nil {
		t.Fatal("expected underlying scaffold stack")
	}
}
