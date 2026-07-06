package kubernetes

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	scaffoldcontainer "github.com/hlfshell/scaffold/container"
)

func (c *Cluster) startSSHBridge(ctx context.Context) error {
	if !c.ssh.enabled {
		return nil
	}

	tempDir, err := os.MkdirTemp("", c.name+"-ssh-*")
	if err != nil {
		return err
	}
	c.sshTempDir = tempDir

	sshDir := filepath.Join(tempDir, "ssh")
	kubeDir := filepath.Join(tempDir, "kube")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		return err
	}
	if err := os.MkdirAll(kubeDir, 0o700); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(sshDir, "authorized_keys"), []byte(strings.Join(c.ssh.authorizedKeys, "\n")+"\n"), 0o600); err != nil {
		return err
	}

	kubeconfig, err := c.internalKubeconfig(ctx)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(kubeDir, "config"), kubeconfig, 0o600); err != nil {
		return err
	}

	container, err := scaffoldcontainer.NewContainer(
		c.name+"-ssh",
		c.ssh.image,
		scaffoldcontainer.WithTag(c.ssh.tag),
		scaffoldcontainer.WithPort("22", c.ssh.hostPort),
		scaffoldcontainer.WithBind(sshDir, "/root/.ssh"),
		scaffoldcontainer.WithBind(kubeDir, "/root/.kube"),
		scaffoldcontainer.WithEntrypoint("sh"),
		scaffoldcontainer.WithCommand("-c", sshBridgeCommand()),
	)
	if err != nil {
		return err
	}
	container.SetLabels(c.labels)
	if c.networkName != "" {
		container.SetNetwork(c.networkName)
	}

	c.sshContainer = container
	if err := c.sshContainer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start kubernetes ssh bridge: %w", err)
	}

	ports := c.sshContainer.GetPorts()
	c.sshPort = ports["22"]

	return nil
}

func (c *Cluster) cleanupSSHBridge(ctx context.Context) error {
	var firstErr error
	if c.sshContainer != nil {
		_, _ = c.sshContainer.Exec(ctx, "sh", "-c", "rm -rf /root/.kube/cache; chmod -R a+rwx /root/.kube /root/.ssh || true")
		if err := c.sshContainer.Cleanup(ctx); err != nil {
			firstErr = err
		}
		c.sshContainer = nil
	}
	if c.sshTempDir != "" {
		if err := os.RemoveAll(c.sshTempDir); err != nil && firstErr == nil {
			firstErr = err
		}
		c.sshTempDir = ""
	}
	c.sshPort = ""

	return firstErr
}

func sshBridgeCommand() string {
	return strings.Join([]string{
		"set -eu",
		"if ! command -v sshd >/dev/null 2>&1; then apk add --no-cache openssh-server >/dev/null; fi",
		"chmod 700 /root/.ssh",
		"chmod 600 /root/.ssh/authorized_keys",
		"mkdir -p /run/sshd /var/run/sshd",
		"ssh-keygen -A >/dev/null 2>&1 || true",
		"printf '\\nPermitRootLogin prohibit-password\\nPubkeyAuthentication yes\\nPasswordAuthentication no\\nStrictModes no\\n' >> /etc/ssh/sshd_config",
		"exec /usr/sbin/sshd -D -e -p 22",
	}, "\n")
}
