package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

/*
Kubectl executes kubectl inside the k3s container and returns combined
stdout and stderr.
*/
func (c *Cluster) Kubectl(ctx context.Context, args ...string) ([]byte, error) {
	command := append([]string{"kubectl"}, args...)
	output, err := c.container.Exec(ctx, command...)
	if err != nil {
		return output, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}

	return output, nil
}

/*
ApplyYAML sends YAML directly to kubectl inside the k3s container.
*/
func (c *Cluster) ApplyYAML(ctx context.Context, yaml []byte) ([]byte, error) {
	return c.applyReader(ctx, bytes.NewReader(yaml))
}

/*
DeleteYAML sends YAML directly to kubectl delete inside the k3s container.
Missing resources are ignored.
*/
func (c *Cluster) DeleteYAML(ctx context.Context, yaml []byte) ([]byte, error) {
	return c.deleteReader(ctx, bytes.NewReader(yaml))
}

/*
ApplyFile reads a manifest file from the host and sends it to kubectl
inside the k3s container.
*/
func (c *Cluster) ApplyFile(ctx context.Context, path string) ([]byte, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return c.ApplyYAML(ctx, contents)
}

/*
ApplyFiles reads host manifest files and applies them in order.
*/
func (c *Cluster) ApplyFiles(ctx context.Context, paths ...string) ([]byte, error) {
	output := []byte{}
	for _, path := range paths {
		result, err := c.ApplyFile(ctx, path)
		output = append(output, result...)
		if err != nil {
			return output, err
		}
	}

	return output, nil
}

/*
Status returns common cluster objects in the configured namespace.
*/
func (c *Cluster) Status(ctx context.Context) ([]byte, error) {
	output := []byte{}
	nodes, err := c.Kubectl(ctx, "get", "nodes")
	output = append(output, nodes...)
	if err != nil {
		return output, err
	}

	args := []string{"get", "pods,svc,deploy,statefulset,daemonset"}
	if c.namespace != "" {
		args = append(args, "-n", c.namespace)
	}

	resources, err := c.Kubectl(ctx, args...)
	if len(output) > 0 && len(resources) > 0 {
		output = append(output, '\n')
	}
	output = append(output, resources...)

	return output, err
}

/*
SSHAddress returns the local SSH address when WithSSH is enabled and the
container has started.
*/
func (c *Cluster) SSHAddress() string {
	if c.sshPort == "" {
		return ""
	}

	return net.JoinHostPort("127.0.0.1", c.sshPort)
}

/*
WaitForRollouts waits for deployment, statefulset, and daemonset
rollouts in the configured namespace.
*/
func (c *Cluster) WaitForRollouts(ctx context.Context) error {
	namespaceArgs := []string{}
	if c.namespace != "" {
		namespaceArgs = []string{"-n", c.namespace}
	}

	for _, kind := range []string{"deployment", "statefulset", "daemonset"} {
		listArgs := append([]string{"get", kind, "-o", "name"}, namespaceArgs...)
		output, err := c.Kubectl(ctx, listArgs...)
		if err != nil {
			return nil
		}

		for _, resource := range strings.Fields(string(output)) {
			args := append([]string{"rollout", "status", resource, "--timeout", c.rolloutTimeout.String()}, namespaceArgs...)
			if _, err := c.Kubectl(ctx, args...); err != nil {
				return fmt.Errorf("failed waiting for %s rollout: %w", resource, err)
			}
		}
	}

	return nil
}

func (c *Cluster) applyReader(ctx context.Context, reader io.Reader) ([]byte, error) {
	args := []string{"kubectl", "apply", "-f", "-"}
	if c.namespace != "" {
		args = append(args, "-n", c.namespace)
	}

	output, err := c.container.ExecInput(ctx, reader, args...)
	if err != nil {
		return output, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}

	return output, nil
}

func (c *Cluster) deleteReader(ctx context.Context, reader io.Reader) ([]byte, error) {
	args := []string{"kubectl", "delete", "-f", "-", "--ignore-not-found=true"}
	if c.namespace != "" {
		args = append(args, "-n", c.namespace)
	}

	output, err := c.container.ExecInput(ctx, reader, args...)
	if err != nil {
		return output, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}

	return output, nil
}

func (c *Cluster) kubectlInNamespace(ctx context.Context, args ...string) ([]byte, error) {
	if c.namespace != "" {
		args = append(args, "-n", c.namespace)
	}

	return c.Kubectl(ctx, args...)
}
