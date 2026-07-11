package datalake

import "testing"

func TestNewStackBuildsIcebergStack(t *testing.T) {
	stack, err := NewStack("scaffold-test-lake", WithBucket("warehouse"))
	if err != nil {
		t.Fatal(err)
	}

	if stack.Lake == nil {
		t.Fatal("expected iceberg stack")
	}
	if stack.Lake.Warehouse == nil {
		t.Fatal("expected warehouse service")
	}
	if stack.Lake.Catalog == nil {
		t.Fatal("expected catalog service")
	}
	if stack.Lake.Trino == nil {
		t.Fatal("expected trino service")
	}
}
