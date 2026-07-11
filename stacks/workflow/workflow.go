package workflow

import (
	"context"
	"time"

	"github.com/hlfshell/scaffold"
	workflows "github.com/hlfshell/scaffold-toolbox/argo-workflows"
	"github.com/hlfshell/scaffold-toolbox/kubernetes"
	"github.com/hlfshell/scaffold/logs"
)

/*
Stack starts a Docker-backed Kubernetes cluster with Argo
Workflows installed. It is meant for local workflow authoring, controller
tests, and image-backed workflow experiments.
*/
type Stack struct {
	Stack     *scaffold.Stack
	Workflows *workflows.Stack
	name      string
}

// Option configures the workflow stack.
type Option func(*workflowConfig)

type workflowConfig struct {
	options []workflows.Option
}

/*
WithNamespace sets the namespace where Argo Workflows is
installed.
*/
func WithNamespace(namespace string) Option {
	return func(config *workflowConfig) {
		config.options = append(config.options, workflows.WithNamespace(namespace))
	}
}

/*
WithManifest applies a Workflow, WorkflowTemplate, or related
manifest after Argo Workflows is ready.
*/
func WithManifest(path string) Option {
	return func(config *workflowConfig) {
		config.options = append(config.options, workflows.WithWorkflowManifest(path))
	}
}

/*
WithKubeconfigPath writes a host kubeconfig for the local
cluster.
*/
func WithKubeconfigPath(path string) Option {
	return func(config *workflowConfig) {
		config.options = append(config.options, workflows.WithKubeconfigPath(path))
	}
}

/*
WithRegistry starts a local registry for workflow images.
*/
func WithRegistry(hostPort string) Option {
	return func(config *workflowConfig) {
		config.options = append(config.options, workflows.WithRegistry(hostPort))
	}
}

/*
WithDockerfileImage builds a Dockerfile and pushes it into the
cluster registry before manifests are applied.
*/
func WithDockerfileImage(dockerfile string, clusterImage string) Option {
	return func(config *workflowConfig) {
		config.options = append(config.options, workflows.WithDockerfileImage(dockerfile, clusterImage))
	}
}

/*
WithLocalImage pushes an existing local image into the cluster
registry before manifests are applied.
*/
func WithLocalImage(localImage string, clusterImage string) Option {
	return func(config *workflowConfig) {
		config.options = append(config.options, workflows.WithLocalImage(localImage, clusterImage))
	}
}

/*
WithSSH exposes SSH into the Kubernetes control-plane container.
*/
func WithSSH(hostPort string, publicKeys ...string) Option {
	return func(config *workflowConfig) {
		config.options = append(config.options, workflows.WithSSH(hostPort, publicKeys...))
	}
}

/*
WithReadyTimeout sets the timeout for cluster readiness.
*/
func WithReadyTimeout(timeout time.Duration) Option {
	return func(config *workflowConfig) {
		config.options = append(config.options, workflows.WithReadyTimeout(timeout))
	}
}

/*
WithClusterOptions passes options to the underlying Kubernetes
cluster.
*/
func WithClusterOptions(options ...kubernetes.Option) Option {
	return func(config *workflowConfig) {
		config.options = append(config.options, workflows.WithClusterOptions(options...))
	}
}

/*
WithOwnedCRDCleanup removes CRDs created by this run during
cleanup. If beforeInstall is true, owned CRDs are also cleared before
install.
*/
func WithOwnedCRDCleanup(beforeInstall bool) Option {
	return func(config *workflowConfig) {
		config.options = append(config.options, workflows.WithOwnedCRDCleanup(beforeInstall))
	}
}

/*
NewStack creates an Argo Workflows quickstart stack.
*/
func NewStack(name string, options ...Option) (*Stack, error) {
	config := &workflowConfig{}
	for _, option := range options {
		option(config)
	}

	workflow, err := workflows.NewStack(name+"-workflows", config.options...)
	if err != nil {
		return nil, err
	}

	stack := &Stack{name: name, Workflows: workflow}
	stack.Stack = scaffold.NewStack(name, scaffold.WithServices(stack.Workflows))

	return stack, nil
}

func (w *Stack) Name() string {
	return w.name
}

/*
SetLabels passes inherited labels to the underlying scaffold stack.
*/
func (w *Stack) SetLabels(labels map[string]string) {
	w.Stack.SetLabels(labels)
}

/*
SetNamePrefix passes an inherited Docker name prefix to the underlying
scaffold stack.
*/
func (w *Stack) SetNamePrefix(prefix string) {
	w.Stack.SetNamePrefix(prefix)
}

/*
Create starts the Kubernetes cluster and installs Argo Workflows.
*/
func (w *Stack) Create(ctx context.Context) error {
	return w.Stack.Create(ctx)
}

/*
IsRunning reports whether any labeled resources for this stack are
running.
*/
func (w *Stack) IsRunning(ctx context.Context) (bool, error) {
	return w.Stack.IsRunning(ctx)
}

/*
Resources returns Docker resources discovered for this stack.
*/
func (w *Stack) Resources(ctx context.Context) (scaffold.ResourceStatus, error) {
	return w.Stack.Resources(ctx)
}

/*
Env returns environment variables exposed by the workflow stack.
*/
func (w *Stack) Env() map[string]string {
	return w.Stack.Env()
}

/*
Endpoints returns endpoints exposed by the workflow stack.
*/
func (w *Stack) Endpoints() map[string]string {
	return w.Stack.Endpoints()
}

/*
Kubectl runs kubectl against the local cluster.
*/
func (w *Stack) Kubectl(ctx context.Context, args ...string) ([]byte, error) {
	return w.Workflows.Kubectl(ctx, args...)
}

/*
Status returns a kubectl summary of cluster resources.
*/
func (w *Stack) Status(ctx context.Context) ([]byte, error) {
	return w.Workflows.Status(ctx)
}

/*
WriteKubeconfig writes a host kubeconfig for the local cluster.
*/
func (w *Stack) WriteKubeconfig(ctx context.Context, path string) (string, error) {
	return w.Workflows.WriteKubeconfig(ctx, path)
}

/*
RegistryAddress returns the host-reachable local registry address.
*/
func (w *Stack) RegistryAddress() string {
	return w.Workflows.RegistryAddress()
}

/*
RegistryImage returns an image reference routed through the local
registry.
*/
func (w *Stack) RegistryImage(image string) string {
	return w.Workflows.RegistryImage(image)
}

/*
RegistryEnv returns environment variables for image publishing helpers.
*/
func (w *Stack) RegistryEnv() map[string]string {
	return w.Workflows.RegistryEnv()
}

/*
PushImage tags and pushes a local image into the workflow registry.
*/
func (w *Stack) PushImage(ctx context.Context, localImage string, clusterImage string) (kubernetes.PushedImage, error) {
	return w.Workflows.PushImage(ctx, localImage, clusterImage)
}

/*
BuildAndPushImage builds a Dockerfile and pushes it into the workflow
registry.
*/
func (w *Stack) BuildAndPushImage(ctx context.Context, dockerfile string, clusterImage string) (kubernetes.PushedImage, string, error) {
	return w.Workflows.BuildAndPushImage(ctx, dockerfile, clusterImage)
}

/*
Cleanup removes resources created by the workflow stack.
*/
func (w *Stack) Cleanup(ctx context.Context) error {
	return w.Stack.Cleanup(ctx)
}

/*
Logs returns logs from the workflow stack services.
*/
func (w *Stack) Logs(ctx context.Context) (logs.LogStreams, error) {
	return w.Stack.Logs(ctx)
}

var _ scaffold.Service = (*Stack)(nil)
var _ scaffold.LabelAttachable = (*Stack)(nil)
var _ scaffold.NamePrefixAttachable = (*Stack)(nil)
