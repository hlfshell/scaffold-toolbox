package kubernetes

import (
	"context"
	"fmt"
	"os"
	"strings"
)

/*
Kubeconfig returns a host-usable kubeconfig for this cluster without
writing it to disk.
*/
func (c *Cluster) Kubeconfig(ctx context.Context) ([]byte, error) {
	output, err := c.container.Exec(ctx, "cat", "/etc/rancher/k3s/k3s.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to read kubeconfig from k3s: %w", err)
	}

	kubeconfig := rewriteServer(string(output), "https://127.0.0.1:"+c.port)
	return []byte(kubeconfig), nil
}

func (c *Cluster) internalKubeconfig(ctx context.Context) ([]byte, error) {
	output, err := c.container.Exec(ctx, "cat", "/etc/rancher/k3s/k3s.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to read kubeconfig from k3s: %w", err)
	}

	kubeconfig := rewriteServer(string(output), "https://"+c.container.Name()+":6443")
	lines := strings.Split(kubeconfig, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.Contains(line, "certificate-authority-data:") {
			filtered = append(filtered, "    insecure-skip-tls-verify: true")
			continue
		}
		filtered = append(filtered, line)
	}

	return []byte(strings.Join(filtered, "\n")), nil
}

func rewriteServer(kubeconfig string, server string) string {
	lines := strings.Split(kubeconfig, "\n")
	for index, line := range lines {
		if strings.Contains(line, "server: https://") {
			prefix := line[:strings.Index(line, "server:")]
			lines[index] = prefix + "server: " + server
		}
	}

	return strings.Join(lines, "\n")
}

/*
WriteKubeconfig writes a host-usable kubeconfig to path. If path is
blank, the path configured by WithKubeconfigPath is used. If neither is
set, a temporary file is created.
*/
func (c *Cluster) WriteKubeconfig(ctx context.Context, path string) (string, error) {
	kubeconfig, err := c.Kubeconfig(ctx)
	if err != nil {
		return "", err
	}

	if path == "" {
		path = c.kubeconfigPath
	}
	if path == "" {
		file, err := os.CreateTemp("", c.name+"-kubeconfig-*")
		if err != nil {
			return "", err
		}
		path = file.Name()
		if err := file.Close(); err != nil {
			return "", err
		}
	}

	if err := os.WriteFile(path, kubeconfig, 0o600); err != nil {
		return "", err
	}
	c.kubeconfigPath = path

	return path, nil
}

/*
KubeconfigPath returns the last path written by WriteKubeconfig, or the
configured default path if WriteKubeconfig has not been called yet.
*/
func (c *Cluster) KubeconfigPath() string {
	return c.kubeconfigPath
}
