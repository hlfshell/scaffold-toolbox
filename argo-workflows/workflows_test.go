package workflows

import (
	"context"
	"os"
	"testing"
	"time"

	scaffoldcontainer "github.com/hlfshell/scaffold/container"
)

func TestArgoWorkflowsCreateStatusCleanup(t *testing.T) {
	if os.Getenv("SCAFFOLD_TOOLBOX_ARGO_TESTS") != "1" {
		t.Skip("set SCAFFOLD_TOOLBOX_ARGO_TESTS=1 to run Argo Workflows integration tests")
	}
	if !scaffoldcontainer.DockerAvailable() {
		t.Skip("docker is not available")
	}

	ctx := context.Background()
	stack, err := NewStack("scaffold-test-workflows",
		WithK3sTag("v1.30.6-k3s1"),
		WithReadyTimeout(3*time.Minute),
		WithRolloutTimeout(6*time.Minute),
		WithOwnedCRDCleanup(true),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := stack.Create(ctx); err != nil {
		t.Fatal(err)
	}
	defer stack.Cleanup(ctx)

	if _, err := stack.Kubectl(ctx, "get", "crd", "workflows.argoproj.io"); err != nil {
		t.Fatal(err)
	}
}
