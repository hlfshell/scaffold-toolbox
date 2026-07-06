package kubernetes

import (
	"fmt"
	"time"

	"github.com/hlfshell/scaffold"
	scaffoldcontainer "github.com/hlfshell/scaffold/container"
)

/*
Cluster is a Docker-backed Kubernetes quickstart. It runs k3s in a
privileged container and applies user-provided YAML through kubectl
inside the container.
*/
type Cluster struct {
	container      *scaffoldcontainer.Container
	sshContainer   *scaffoldcontainer.Container
	name           string
	tag            string
	port           string
	namespace      string
	kubeconfigPath string
	networkName    string
	networkCreated bool
	manifests      []manifest
	sshPort        string
	labels         map[string]string
	k3sArgs        []string
	ssh            sshConfig
	sshTempDir     string
	readyTimeout   time.Duration
	rolloutTimeout time.Duration
}

type sshConfig struct {
	enabled        bool
	hostPort       string
	image          string
	tag            string
	authorizedKeys []string
}

// Option configures the Kubernetes cluster before the container is built.
type Option func(*Cluster)

/*
NewCluster creates a k3s-backed Kubernetes cluster service.
*/
func NewCluster(name string, options ...Option) (*Cluster, error) {
	cluster := &Cluster{
		name:   name,
		tag:    "latest",
		labels: map[string]string{},
		ssh: sshConfig{
			image: "alpine/k8s",
			tag:   "1.30.6",
		},
		readyTimeout:   2 * time.Minute,
		rolloutTimeout: 2 * time.Minute,
	}
	for _, option := range options {
		option(cluster)
	}
	if cluster.ssh.enabled && len(cluster.ssh.authorizedKeys) == 0 {
		return nil, fmt.Errorf("kubernetes SSH requires at least one authorized public key")
	}

	containerOptions := []scaffoldcontainer.ContainerOption{
		scaffoldcontainer.WithTag(cluster.tag),
		scaffoldcontainer.WithPort("6443", ""),
		scaffoldcontainer.WithPrivileged(),
		scaffoldcontainer.WithCommand(cluster.command()...),
	}

	if err := cluster.prepareManifests(); err != nil {
		return nil, err
	}
	for _, manifest := range cluster.manifests {
		if manifest.bind {
			containerOptions = append(containerOptions, scaffoldcontainer.WithBind(manifest.source, manifest.containerPath))
		}
	}

	container, err := scaffoldcontainer.NewContainer(name, "rancher/k3s", containerOptions...)
	if err != nil {
		return nil, err
	}
	cluster.container = container

	return cluster, nil
}

func (c *Cluster) Name() string {
	return c.name
}

func (c *Cluster) SetLabels(labels map[string]string) {
	c.labels = merge(c.labels, labels)
	c.container.SetLabels(labels)
}

func (c *Cluster) SetNamePrefix(prefix string) {
	c.container.SetNamePrefix(prefix)
}

func (c *Cluster) SetNetwork(name string) {
	c.networkName = name
	c.container.SetNetwork(name)
	if c.sshContainer != nil {
		c.sshContainer.SetNetwork(name)
	}
}

func (c *Cluster) command() []string {
	args := []string{
		"server",
		"--disable", "traefik",
		"--tls-san", "127.0.0.1",
		"--bind-address", "0.0.0.0",
		"--https-listen-port", "6443",
	}
	args = append(args, c.k3sArgs...)
	return args
}

func merge(left map[string]string, right map[string]string) map[string]string {
	output := map[string]string{}
	for key, value := range left {
		output[key] = value
	}
	for key, value := range right {
		output[key] = value
	}

	return output
}

var _ scaffold.Service = (*Cluster)(nil)
var _ scaffold.LabelAttachable = (*Cluster)(nil)
var _ scaffold.NamePrefixAttachable = (*Cluster)(nil)
var _ scaffold.NetworkAttachable = (*Cluster)(nil)
