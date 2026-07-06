package iceberg

import "testing"

func TestNewStackBuildsComposedServices(t *testing.T) {
	stack, err := NewStack("scaffold-test-iceberg", WithBucket("warehouse"))
	if err != nil {
		t.Fatal(err)
	}

	if stack.Warehouse == nil {
		t.Fatal("expected warehouse service")
	}
	if stack.Catalog == nil {
		t.Fatal("expected catalog service")
	}
	if stack.Trino == nil {
		t.Fatal("expected trino service")
	}
	if stack.Stack == nil {
		t.Fatal("expected underlying scaffold stack")
	}
}
