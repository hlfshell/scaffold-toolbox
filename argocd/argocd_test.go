package argocd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hlfshell/scaffold"
	scaffoldcontainer "github.com/hlfshell/scaffold/container"
)

func TestArgoCDCreateStatusCleanup(t *testing.T) {
	if os.Getenv("SCAFFOLD_TOOLBOX_ARGO_TESTS") != "1" {
		t.Skip("set SCAFFOLD_TOOLBOX_ARGO_TESTS=1 to run Argo CD integration tests")
	}
	if !scaffoldcontainer.DockerAvailable() {
		t.Skip("docker is not available")
	}

	ctx := context.Background()
	dir := t.TempDir()
	dockerfile := filepath.Join(dir, "Dockerfile")
	if err := os.WriteFile(dockerfile, []byte(`FROM busybox:1.36
CMD ["sh", "-c", "while true; do sleep 3600; done"]
`), 0o600); err != nil {
		t.Fatal(err)
	}

	stack, err := NewStack("scaffold-test-argocd",
		WithK3sTag("v1.30.6-k3s1"),
		WithRegistry(""),
		WithDockerfileImage(dockerfile, "scaffold/argocd-registry:latest"),
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

	if _, err := stack.Kubectl(ctx, "get", "crd", "applications.argoproj.io"); err != nil {
		t.Fatal(err)
	}

	registryApp := argoCDApplication("registry-image", "argocd-registry", stack.RegistryImage("scaffold/argocd-registry:latest"))
	if output, err := stack.cluster.ApplyYAML(ctx, registryApp); err != nil {
		t.Fatalf("failed to apply registry application: %v: %s", err, string(output))
	}
	waitForArgoCDApplication(t, ctx, stack, "registry-image")
	waitForDeploymentImage(t, ctx, stack, stack.RegistryImage("scaffold/argocd-registry:latest"))

	publicApp := argoCDApplication("public-image", "argocd-public", "")
	if output, err := stack.cluster.ApplyYAML(ctx, publicApp); err != nil {
		t.Fatalf("failed to apply public application: %v: %s", err, string(output))
	}
	waitForArgoCDApplication(t, ctx, stack, "public-image")
}

func argoCDApplication(name string, namespace string, image string) []byte {
	kustomize := ""
	if image != "" {
		kustomize = `
    kustomize:
      images:
        - gcr.io/google-samples/gb-frontend:v5=` + image
	}

	return []byte(`apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: ` + name + `
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/argoproj/argocd-example-apps.git
    targetRevision: HEAD
    path: kustomize-guestbook` + kustomize + `
  destination:
    server: https://kubernetes.default.svc
    namespace: ` + namespace + `
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
`)
}

func waitForArgoCDApplication(t *testing.T, ctx context.Context, stack *Stack, name string) {
	t.Helper()

	err := scaffold.WaitFunc(ctx, 4*time.Minute, 2*time.Second, func(ctx context.Context) error {
		output, err := stack.Kubectl(ctx, "get", "application", name, "-n", "argocd", "-o", "jsonpath={.status.sync.status}/{.status.health.status}")
		if err != nil {
			return err
		}
		if string(output) != "Synced/Healthy" {
			return fmt.Errorf("application %s is %s", name, string(output))
		}

		return nil
	})
	if err != nil {
		status, _ := stack.Kubectl(ctx, "get", "application", name, "-n", "argocd", "-o", "yaml")
		t.Fatalf("application %s did not sync and become healthy: %v\n%s", name, err, string(status))
	}
}

func waitForDeploymentImage(t *testing.T, ctx context.Context, stack *Stack, image string) {
	t.Helper()

	err := scaffold.WaitFunc(ctx, 2*time.Minute, 2*time.Second, func(ctx context.Context) error {
		output, err := stack.Kubectl(ctx, "get", "deployments", "-A", "-o", "jsonpath={range .items[*]}{.metadata.namespace}/{.metadata.name}={.spec.template.spec.containers[*].image}{\"\\n\"}{end}")
		if err != nil {
			return err
		}
		if !strings.Contains(string(output), image) {
			return fmt.Errorf("deployment image %s not found in %s", image, string(output))
		}

		return nil
	})
	if err != nil {
		app, _ := stack.Kubectl(ctx, "get", "application", "registry-image", "-n", "argocd", "-o", "yaml")
		deployments, _ := stack.Kubectl(ctx, "get", "deployments", "-A", "-o", "wide")
		t.Fatalf("registry deployment image was not used: %v\napplication:\n%s\ndeployments:\n%s", err, string(app), string(deployments))
	}
}
