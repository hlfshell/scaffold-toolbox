package workflows

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hlfshell/scaffold"
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
	dir := t.TempDir()
	dockerfile := filepath.Join(dir, "Dockerfile")
	if err := os.WriteFile(dockerfile, []byte(`FROM busybox:1.36
CMD ["sh", "-c", "echo workflow-registry-image"]
`), 0o600); err != nil {
		t.Fatal(err)
	}

	stack, err := NewStack("scaffold-test-workflows",
		WithK3sTag("v1.30.6-k3s1"),
		WithRegistry(""),
		WithDockerfileImage(dockerfile, "scaffold/workflow-registry:latest"),
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

	registryWorkflow := workflowManifest("registry-image", stack.RegistryImage("scaffold/workflow-registry:latest"))
	if output, err := stack.cluster.ApplyYAML(ctx, registryWorkflow); err != nil {
		t.Fatalf("failed to apply registry workflow: %v: %s", err, string(output))
	}
	waitForWorkflowPhase(t, ctx, stack, "registry-image", "Succeeded")

	podImage, err := stack.Kubectl(ctx, "get", "workflow", "registry-image", "-n", "argo", "-o", "jsonpath={.spec.templates[0].container.image}")
	if err != nil {
		t.Fatal(err)
	}
	if string(podImage) != stack.RegistryImage("scaffold/workflow-registry:latest") {
		t.Fatalf("expected workflow to use registry image, got %s", string(podImage))
	}

	publicWorkflow := workflowManifest("public-image", "busybox:1.36")
	if output, err := stack.cluster.ApplyYAML(ctx, publicWorkflow); err != nil {
		t.Fatalf("failed to apply public workflow: %v: %s", err, string(output))
	}
	waitForWorkflowPhase(t, ctx, stack, "public-image", "Succeeded")
}

func workflowManifest(name string, image string) []byte {
	return []byte(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: ` + name + `
  namespace: argo
spec:
  entrypoint: main
  templates:
    - name: main
      container:
        image: ` + image + `
        command: ["sh", "-c"]
        args: ["echo ` + name + `"]
`)
}

func waitForWorkflowPhase(t *testing.T, ctx context.Context, stack *Stack, name string, phase string) {
	t.Helper()

	err := scaffold.WaitFunc(ctx, 4*time.Minute, 2*time.Second, func(ctx context.Context) error {
		output, err := stack.Kubectl(ctx, "get", "workflow", name, "-n", "argo", "-o", "jsonpath={.status.phase}")
		if err != nil {
			return err
		}
		if string(output) != phase {
			return fmt.Errorf("workflow %s is %s", name, string(output))
		}

		return nil
	})
	if err != nil {
		status, _ := stack.Kubectl(ctx, "get", "workflow", name, "-n", "argo", "-o", "yaml")
		t.Fatalf("workflow %s did not reach %s: %v\n%s", name, phase, err, string(status))
	}
}
