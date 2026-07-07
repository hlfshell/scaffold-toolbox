# scaffold toolbox argocd

Argo CD quickstart for scaffold. It starts a Docker-backed k3s cluster,
installs Argo CD, waits for its CRDs/control plane, and then applies
Application or AppProject manifests.

## Install

```bash
go get github.com/hlfshell/scaffold-toolbox/argocd
```

```go
import "github.com/hlfshell/scaffold-toolbox/argocd"
```

## Example

```go
gitops, err := argocd.NewStack("gitops",
	argocd.WithNamespace("argocd"),
	argocd.WithApplicationManifest("./apps"),
	argocd.WithOwnedCRDCleanup(true),
)
if err != nil {
	return err
}
```

`WithOwnedCRDCleanup(true)` records whether Argo CD CRDs existed before the
stack starts. Cleanup only removes CRDs that were created by this run. If the
CRDs were already present, scaffold leaves them alone. The `true` value also
clears owned CRDs before install, which is useful after a failed local run.

Common Kubernetes quickstart options are passed through:

```go
gitops, err := argocd.NewStack("gitops",
	argocd.WithK3sTag("v1.30.6-k3s1"),
	argocd.WithReadyTimeout(4*time.Minute),
	argocd.WithRolloutTimeout(6*time.Minute),
	argocd.WithSSH("", string(publicKey)),
)
```

The local image registry options are also passed through:

```go
gitops, err := argocd.NewStack("gitops",
	argocd.WithRegistry(""),
	argocd.WithDockerfileImage("./Dockerfile", "app/api:dev"),
)
```

After startup, use `gitops.RegistryImage("app/api:dev")` in generated
Application manifests or call `PushImage` / `BuildAndPushImage` for later
updates.
