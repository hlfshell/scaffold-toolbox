package kubernetes

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	scaffoldcontainer "github.com/hlfshell/scaffold/container"
)

func TestClusterCreateApplyStatusKubeconfigCleanup(t *testing.T) {
	if os.Getenv("SCAFFOLD_TOOLBOX_KUBERNETES_TESTS") != "1" {
		t.Skip("set SCAFFOLD_TOOLBOX_KUBERNETES_TESTS=1 to run k3s integration tests")
	}
	if !scaffoldcontainer.DockerAvailable() {
		t.Skip("docker is not available")
	}

	ctx := context.Background()
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "configmap.yaml")
	if err := os.WriteFile(manifestPath, []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: scaffold-test
data:
  hello: world
`), 0o600); err != nil {
		t.Fatal(err)
	}
	dockerfilePath := filepath.Join(dir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(`FROM busybox:1.36
CMD ["sh", "-c", "while true; do sleep 3600; done"]
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cluster, err := NewCluster("scaffold-test-kubernetes",
		WithTag("v1.30.6-k3s1"),
		WithNamespace("default"),
		WithRegistry(""),
		WithDockerfileImage(dockerfilePath, "scaffold/demo:latest"),
		WithManifest(manifestPath),
		WithReadyTimeout(3*time.Minute),
		WithRolloutTimeout(2*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := cluster.Create(ctx); err != nil {
		t.Fatal(err)
	}
	defer cluster.Cleanup(ctx)

	status, err := cluster.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(status), "Ready") {
		t.Fatalf("expected ready node in status, got %s", string(status))
	}

	output, err := cluster.ApplyYAML(ctx, []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: scaffold-test-inline
data:
  hello: inline
`))
	if err != nil {
		t.Fatalf("apply inline manifest failed: %v: %s", err, string(output))
	}

	kubeconfigPath := filepath.Join(dir, "kubeconfig")
	if _, err := cluster.WriteKubeconfig(ctx, kubeconfigPath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(kubeconfigPath); err != nil {
		t.Fatal(err)
	}

	if cluster.RegistryAddress() == "" {
		t.Fatal("expected host registry address")
	}
	if cluster.RegistryInternalAddress() == "" {
		t.Fatal("expected internal registry address")
	}
	if _, err := cluster.RegistryDockerConfigJSON(); err != nil {
		t.Fatal(err)
	}

	pod := []byte(`apiVersion: v1
kind: Pod
metadata:
  name: scaffold-registry-test
spec:
  restartPolicy: Never
  containers:
    - name: app
      image: ` + cluster.RegistryImage("scaffold/demo:latest") + `
      imagePullPolicy: Always
`)
	if output, err := cluster.ApplyYAML(ctx, pod); err != nil {
		t.Fatalf("apply registry pod failed: %v: %s", err, string(output))
	}
	if output, err := cluster.Kubectl(ctx, "wait", "--for=condition=Ready", "pod/scaffold-registry-test", "--timeout=2m"); err != nil {
		t.Fatalf("registry pod did not become ready: %v: %s", err, string(output))
	}
}
