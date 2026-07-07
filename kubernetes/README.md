# scaffold toolbox kubernetes

**Warning** - this tool requires running k3s in a privileged container. This is useful for local experiments, but it is a broad Docker permission. Use it only where that tradeoff is acceptable.

`scaffold`'s Kubernetes tooling is built around the `rancher/k3s` image. You'll primarily interact with it through YAML manifest files. The service waits for common rollouts and exposes a kubectl passthrough for interactive work.

`Kubeconfig` or `WriteKubeconfig` gets you a Kubernetes config that points host tools at the containerized k3s API. The former returns the config bytes; the latter writes them to a file. Neither is called automatically.

You can enable a hosted Docker registry to push your own container images or Dockerfiles onto the cluster for reference by your manifest files. See [Local image registry](#local-image-registry) for more information on this.

## Install

```bash
go get github.com/hlfshell/scaffold-toolbox/kubernetes
```

```go
import "github.com/hlfshell/scaffold-toolbox/kubernetes"
```

## Example

```go
package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

	"github.com/hlfshell/scaffold"
	"github.com/hlfshell/scaffold-toolbox/kubernetes"
)

//go:embed deploy/*.yaml
var deployFS embed.FS

func main() {
	ctx := context.Background()

	cluster, err := kubernetes.NewCluster("cluster",
		kubernetes.WithNamespace("dev"),
		kubernetes.WithManifest("https://raw.githubusercontent.com/kubernetes/website/main/content/en/examples/application/deployment.yaml"),
	)
	if err != nil {
		panic(err)
	}

	stack := scaffold.NewStack("dev", scaffold.WithServices(cluster))
	if err := stack.Create(ctx); err != nil {
		panic(err)
	}
	defer stack.Cleanup(ctx)

	if err := applyEmbedded(ctx, cluster, deployFS, "deploy"); err != nil {
		panic(err)
	}

	kubeconfig, err := cluster.WriteKubeconfig(ctx, "./kubeconfig.dev")
	if err != nil {
		panic(err)
	}
	fmt.Printf("KUBECONFIG=%s\n", kubeconfig)

	status, err := cluster.Status(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(status))

	logs, err := cluster.Kubectl(ctx, "logs", "deploy/api", "-n", "dev")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(logs))
}

func applyEmbedded(ctx context.Context, cluster *kubernetes.Cluster, files embed.FS, root string) error {
	return fs.WalkDir(files, root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}

		contents, err := files.ReadFile(path)
		if err != nil {
			return err
		}

		_, err = cluster.ApplyYAML(ctx, contents)
		return err
	})
}
```

For host files that are not embedded:

```go
_, err := cluster.ApplyFiles(ctx, "./deploy/api.yaml", "./deploy/service.yaml")
```

For direct YAML:

```go
_, err := cluster.ApplyYAML(ctx, []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: demo
data:
  hello: world
`))
```

For local kubectl after startup:

```bash
KUBECONFIG=./kubeconfig.dev kubectl get pods -n dev
```

## Local image registry

`WithRegistry` starts a local Docker registry on the same Docker network as
the cluster and configures k3s to pull from it over the internal registry
address.

```go
cluster, err := kubernetes.NewCluster("cluster",
	kubernetes.WithNamespace("dev"),
	kubernetes.WithRegistry(""),
	kubernetes.WithDockerfileImage("./Dockerfile", "app/api:dev"),
)
if err != nil {
	return err
}
```

After `Create`, use `RegistryImage` when writing or patching manifests:

```go
image := cluster.RegistryImage("app/api:dev")
```

The host Docker daemon pushes to `RegistryAddress`, while Kubernetes pulls
from `RegistryInternalAddress`:

```go
fmt.Println(cluster.RegistryAddress())
fmt.Println(cluster.RegistryInternalAddress())
fmt.Println(cluster.RegistryEnv())
```

You can also push images after the cluster is running:

```go
pushed, err := cluster.PushImage(ctx, "api:local", "app/api:dev")
if err != nil {
	return err
}
fmt.Println(pushed.ClusterImage)

pushed, logs, err := cluster.BuildAndPushImage(ctx, "./Dockerfile", "app/worker:dev")
if err != nil {
	fmt.Println(logs)
	return err
}
```

`RegistryDockerConfigJSON` returns a host-side Docker `config.json` payload
for tools that expect one. The default registry has no username or password.

To expose SSH, pass at least one authorized public key. This starts a companion SSH/kubectl container on the same Docker network as k3s. It does not SSH into the `rancher/k3s` container itself; a companion container gets a generated kubeconfig mounted into `/root/.kube/config`, so you can SSH in and run `kubectl` against the k3s cluster.

```go
key, err := os.ReadFile("/home/me/.ssh/id_ed25519.pub")
if err != nil {
	return err
}

cluster, err := kubernetes.NewCluster("cluster",
	kubernetes.WithNamespace("dev"),
	kubernetes.WithSSH("2222", string(key)),
)
if err != nil {
	return err
}
```

After startup:

```bash
ssh -p 2222 root@127.0.0.1
```

If you leave the host port blank, Docker assigns one:

```go
cluster, err := kubernetes.NewCluster("cluster",
	kubernetes.WithSSH("", string(key)),
)
```

Then read it with:

```go
fmt.Println(cluster.SSHAddress())
```
