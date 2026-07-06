# scaffold toolbox argo-workflows

Argo Workflows quickstart for scaffold. It starts a Docker-backed k3s
cluster, installs Argo Workflows, waits for its CRDs/controller, and then
applies Workflow or WorkflowTemplate manifests.

## Install

```bash
go get github.com/hlfshell/scaffold-toolbox/argo-workflows
```

```go
import "github.com/hlfshell/scaffold-toolbox/argo-workflows"
```

## Example

```go
workflows, err := workflows.NewStack("workflows",
	workflows.WithNamespace("argo"),
	workflows.WithWorkflowManifest("./workflows"),
	workflows.WithOwnedCRDCleanup(true),
)
if err != nil {
	return err
}
```

`WithOwnedCRDCleanup(true)` records whether Argo Workflows CRDs existed before
the stack starts. Cleanup only removes CRDs that were created by this run. If
the CRDs were already present, scaffold leaves them alone. The `true` value
also clears owned CRDs before install, which is useful after a failed local run.

Common Kubernetes quickstart options are passed through:

```go
workflows, err := workflows.NewStack("workflows",
	workflows.WithK3sTag("v1.30.6-k3s1"),
	workflows.WithReadyTimeout(4*time.Minute),
	workflows.WithRolloutTimeout(6*time.Minute),
	workflows.WithSSH("", string(publicKey)),
)
```
