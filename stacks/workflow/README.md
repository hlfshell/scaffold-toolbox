# scaffold toolbox workflow stack

Local workflow authoring stack for scaffold. It starts a Docker-backed Kubernetes cluster, installs Argo Workflows, and can preload workflow manifests and container images.

## Install

```bash
go get github.com/hlfshell/scaffold-toolbox/stacks/workflow
```

```go
import "github.com/hlfshell/scaffold-toolbox/stacks/workflow"
```

## Example

```go
stack, err := workflow.NewStack("workflow-dev",
	workflow.WithKubeconfigPath("kubeconfig.workflow"),
	workflow.WithRegistry(""),
	workflow.WithDockerfileImage("./worker.Dockerfile", "worker:dev"),
	workflow.WithManifest("workflows/example.yaml"),
)
if err != nil {
	return err
}

if err := stack.Create(ctx); err != nil {
	return err
}

_, err = stack.Kubectl(ctx, "get", "workflows", "-n", "argo")
```
