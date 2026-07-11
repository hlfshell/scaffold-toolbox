package rag

import "testing"

func TestNewStackBuildsComposedServices(t *testing.T) {
	stack, err := NewStack("scaffold-test-rag")
	if err != nil {
		t.Fatal(err)
	}

	if stack.Postgres == nil {
		t.Fatal("expected postgres service")
	}
	if stack.Qdrant == nil {
		t.Fatal("expected qdrant service")
	}
	if stack.MinIO == nil {
		t.Fatal("expected minio service")
	}
	if stack.Stack == nil {
		t.Fatal("expected underlying scaffold stack")
	}
}
