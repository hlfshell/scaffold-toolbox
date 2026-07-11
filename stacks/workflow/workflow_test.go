package workflow

import (
	"testing"
	"time"
)

func TestNewStackBuildsArgoStack(t *testing.T) {
	stack, err := NewStack(
		"scaffold-test-workflows",
		WithNamespace("workflows"),
		WithReadyTimeout(2*time.Minute),
		WithRegistry(""),
	)
	if err != nil {
		t.Fatal(err)
	}

	if stack.Workflows == nil {
		t.Fatal("expected argo workflows stack")
	}
	if stack.Stack == nil {
		t.Fatal("expected underlying scaffold stack")
	}
}
