package kubernetes

import "time"

/*
WithTag sets the rancher/k3s image tag. Pin this in CI for repeatable
cluster behavior.
*/
func WithTag(tag string) Option {
	return func(cluster *Cluster) {
		if tag != "" {
			cluster.tag = tag
		}
	}
}

/*
WithNamespace creates and uses a namespace for manifest apply, delete,
status, and rollout helpers.
*/
func WithNamespace(name string) Option {
	return func(cluster *Cluster) {
		cluster.namespace = name
	}
}

/*
WithManifest registers a YAML file, directory, or URL to apply after the
cluster is ready. Local paths are bind-mounted into the k3s container;
URLs are passed directly to kubectl.
*/
func WithManifest(path string) Option {
	return func(cluster *Cluster) {
		cluster.manifests = append(cluster.manifests, manifest{source: path})
	}
}

/*
WithRegistry starts a local Docker registry beside the cluster. hostPort
may be blank to let Docker assign a free host port.
*/
func WithRegistry(hostPort string) Option {
	return func(cluster *Cluster) {
		cluster.registryConfig.enabled = true
		cluster.registryConfig.hostPort = hostPort
	}
}

/*
WithRegistryImage changes the local registry container image.
*/
func WithRegistryImage(image string, tag string) Option {
	return func(cluster *Cluster) {
		cluster.registryConfig.enabled = true
		if image != "" {
			cluster.registryConfig.image = image
		}
		if tag != "" {
			cluster.registryConfig.tag = tag
		}
	}
}

/*
WithLocalImage tags and pushes an existing local Docker image into the
cluster registry before manifests are applied.
*/
func WithLocalImage(localImage string, clusterImage string) Option {
	return func(cluster *Cluster) {
		cluster.registryConfig.enabled = true
		cluster.images = append(cluster.images, Image{
			LocalImage:   localImage,
			ClusterImage: clusterImage,
		})
	}
}

/*
WithDockerfileImage builds the Dockerfile and pushes the result into the
cluster registry before manifests are applied.
*/
func WithDockerfileImage(dockerfile string, clusterImage string) Option {
	return func(cluster *Cluster) {
		cluster.registryConfig.enabled = true
		cluster.images = append(cluster.images, Image{
			Dockerfile:   dockerfile,
			ClusterImage: clusterImage,
		})
	}
}

/*
WithSSH starts a companion SSH/kubectl container configured against the
k3s API. Public keys are written to root's authorized_keys inside that
companion container. hostPort may be blank to let Docker assign a free
port.
*/
func WithSSH(hostPort string, publicKeys ...string) Option {
	return func(cluster *Cluster) {
		cluster.ssh.enabled = true
		cluster.ssh.hostPort = hostPort
		cluster.ssh.authorizedKeys = append(cluster.ssh.authorizedKeys, publicKeys...)
	}
}

/*
WithSSHImage changes the companion SSH/kubectl image. The image must have
kubectl, sh, apk, and OpenSSH packages available or installable.
*/
func WithSSHImage(image string, tag string) Option {
	return func(cluster *Cluster) {
		if image != "" {
			cluster.ssh.image = image
		}
		if tag != "" {
			cluster.ssh.tag = tag
		}
	}
}

/*
WithKubeconfigPath sets the default path used by WriteKubeconfig.
Nothing is written unless WriteKubeconfig is called.
*/
func WithKubeconfigPath(path string) Option {
	return func(cluster *Cluster) {
		cluster.kubeconfigPath = path
	}
}

/*
WithK3sArgs appends arguments to the k3s server command.
*/
func WithK3sArgs(args ...string) Option {
	return func(cluster *Cluster) {
		cluster.k3sArgs = append(cluster.k3sArgs, args...)
	}
}

/*
WithReadyTimeout changes how long Create waits for the Kubernetes API.
*/
func WithReadyTimeout(timeout time.Duration) Option {
	return func(cluster *Cluster) {
		if timeout > 0 {
			cluster.readyTimeout = timeout
		}
	}
}

/*
WithRolloutTimeout changes how long Create waits for deployments,
statefulsets, and daemonsets to finish rolling out after manifests are
applied.
*/
func WithRolloutTimeout(timeout time.Duration) Option {
	return func(cluster *Cluster) {
		if timeout > 0 {
			cluster.rolloutTimeout = timeout
		}
	}
}
