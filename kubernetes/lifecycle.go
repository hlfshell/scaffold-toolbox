package kubernetes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hlfshell/scaffold"
	scaffoldcontainer "github.com/hlfshell/scaffold/container"
	"github.com/hlfshell/scaffold/logs"
)

/*
Create starts the k3s container, waits for the Kubernetes API, applies
registered manifests, and waits for rollouts.
*/
func (c *Cluster) Create(ctx context.Context) error {
	if c.ssh.enabled && c.networkName == "" {
		c.networkName = c.name + "-network"
		created, err := scaffoldcontainer.CreateNetwork(ctx, c.networkName, c.labels)
		if err != nil {
			return err
		}
		c.networkCreated = created
		c.container.SetNetwork(c.networkName)
	}

	err := c.container.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start k3s container: %w", err)
	}

	ports := c.container.GetPorts()
	c.port = ports["6443"]
	c.sshPort = ports["22"]

	err = scaffold.WaitForTCP(ctx, "127.0.0.1", c.port, c.readyTimeout)
	if err != nil {
		c.cleanupAfterCreateFailure(ctx)
		return fmt.Errorf("kubernetes api failed to open: %w", err)
	}

	err = scaffold.WaitFunc(ctx, c.readyTimeout, 500*time.Millisecond, func(ctx context.Context) error {
		output, err := c.Kubectl(ctx, "get", "nodes", "--no-headers")
		if err != nil {
			return err
		}
		if !strings.Contains(string(output), " Ready ") {
			return fmt.Errorf("no ready kubernetes node yet")
		}

		return nil
	})
	if err != nil {
		c.cleanupAfterCreateFailure(ctx)
		return fmt.Errorf("kubernetes cluster failed to become ready: %w", err)
	}

	if err := c.startSSHBridge(ctx); err != nil {
		c.cleanupAfterCreateFailure(ctx)
		return err
	}

	if c.namespace != "" {
		_, err := c.Kubectl(ctx, "create", "namespace", c.namespace)
		if err != nil && !strings.Contains(err.Error(), "AlreadyExists") {
			c.cleanupAfterCreateFailure(ctx)
			return fmt.Errorf("failed to create namespace %s: %w", c.namespace, err)
		}
	}

	for _, manifest := range c.manifests {
		if _, err := c.kubectlInNamespace(ctx, "apply", "-f", manifest.containerPath); err != nil {
			c.cleanupAfterCreateFailure(ctx)
			return fmt.Errorf("failed to apply manifest %s: %w", manifest.source, err)
		}
	}

	if err := c.WaitForRollouts(ctx); err != nil {
		c.cleanupAfterCreateFailure(ctx)
		return err
	}

	return nil
}

/*
Cleanup deletes registered manifests in reverse order and removes the k3s
container.
*/
func (c *Cluster) Cleanup(ctx context.Context) error {
	for i := len(c.manifests) - 1; i >= 0; i-- {
		_, _ = c.kubectlInNamespace(ctx, "delete", "-f", c.manifests[i].containerPath, "--ignore-not-found=true")
	}

	var firstErr error
	if err := c.cleanupSSHBridge(ctx); err != nil {
		firstErr = err
	}
	if err := c.container.Cleanup(ctx); err != nil && firstErr == nil {
		firstErr = err
	}
	if c.networkCreated {
		if err := scaffoldcontainer.RemoveNetwork(ctx, c.networkName); err != nil && firstErr == nil {
			firstErr = err
		}
		c.networkCreated = false
	}

	return firstErr
}

func (c *Cluster) Env() map[string]string {
	env := map[string]string{}
	if c.kubeconfigPath != "" {
		env["KUBECONFIG"] = c.kubeconfigPath
	}
	if c.namespace != "" {
		env["KUBE_NAMESPACE"] = c.namespace
	}
	if c.sshPort != "" {
		env["KUBE_SSH_ADDR"] = c.SSHAddress()
	}

	return env
}

func (c *Cluster) Endpoints() map[string]string {
	endpoints := map[string]string{
		c.name: "https://127.0.0.1:" + c.port,
	}
	if c.sshPort != "" {
		endpoints[c.name+"-ssh"] = c.SSHAddress()
	}

	return endpoints
}

func (c *Cluster) Logs(ctx context.Context) (logs.LogStreams, error) {
	stream, err := c.container.Logs(ctx)
	if err != nil {
		return nil, err
	}

	streams := logs.LogStreams{c.name: stream}
	if c.sshContainer != nil {
		sshStream, err := c.sshContainer.Logs(ctx)
		if err != nil {
			_ = streams.Close()
			return nil, err
		}
		streams[c.name+"-ssh"] = sshStream
	}

	return streams, nil
}

func (c *Cluster) cleanupAfterCreateFailure(ctx context.Context) {
	ctx = context.WithoutCancel(ctx)
	_ = c.cleanupSSHBridge(ctx)
	_ = c.container.Cleanup(ctx)
	if c.networkCreated {
		_ = scaffoldcontainer.RemoveNetwork(ctx, c.networkName)
		c.networkCreated = false
	}
}
