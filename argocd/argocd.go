package argocd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hlfshell/scaffold"
	"github.com/hlfshell/scaffold-toolbox/kubernetes"
	"github.com/hlfshell/scaffold/logs"
)

const installManifest = "https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml"

var crdNames = []string{
	"applications.argoproj.io",
	"applicationsets.argoproj.io",
	"appprojects.argoproj.io",
}

/*
Stack starts a Docker-backed k3s cluster, installs Argo CD, waits for the
CRDs and core deployments, and then applies user application manifests.
*/
type Stack struct {
	cluster     *kubernetes.Cluster
	name        string
	namespace   string
	manifests   []string
	preexisting map[string]bool
	cleanupCRDs bool
	clearCRDs   bool
}

// Option configures the Argo CD stack.
type Option func(*config)

type config struct {
	namespace      string
	kubeconfigPath string
	manifests      []string
	clusterOptions []kubernetes.Option
	cleanupCRDs    bool
	clearCRDs      bool
}

func WithKubeconfigPath(path string) Option {
	return func(config *config) {
		config.kubeconfigPath = path
	}
}

func WithNamespace(name string) Option {
	return func(config *config) {
		if name != "" {
			config.namespace = name
		}
	}
}

/*
WithApplicationManifest registers an Argo CD Application or AppProject
manifest to apply after Argo CD is installed and its CRDs are available.
*/
func WithApplicationManifest(path string) Option {
	return func(config *config) {
		config.manifests = append(config.manifests, path)
	}
}

/*
WithClusterOptions passes options to the underlying Kubernetes quickstart.
Use this for k3s args, timeouts, SSH, and other cluster-level behavior.
*/
func WithClusterOptions(options ...kubernetes.Option) Option {
	return func(config *config) {
		config.clusterOptions = append(config.clusterOptions, options...)
	}
}

func WithK3sTag(tag string) Option {
	return WithClusterOptions(kubernetes.WithTag(tag))
}

func WithK3sArgs(args ...string) Option {
	return WithClusterOptions(kubernetes.WithK3sArgs(args...))
}

func WithSSH(hostPort string, publicKeys ...string) Option {
	return WithClusterOptions(kubernetes.WithSSH(hostPort, publicKeys...))
}

func WithRegistry(hostPort string) Option {
	return WithClusterOptions(kubernetes.WithRegistry(hostPort))
}

func WithRegistryImage(image string, tag string) Option {
	return WithClusterOptions(kubernetes.WithRegistryImage(image, tag))
}

func WithLocalImage(localImage string, clusterImage string) Option {
	return WithClusterOptions(kubernetes.WithLocalImage(localImage, clusterImage))
}

func WithDockerfileImage(dockerfile string, clusterImage string) Option {
	return WithClusterOptions(kubernetes.WithDockerfileImage(dockerfile, clusterImage))
}

func WithReadyTimeout(timeout time.Duration) Option {
	return WithClusterOptions(kubernetes.WithReadyTimeout(timeout))
}

func WithRolloutTimeout(timeout time.Duration) Option {
	return WithClusterOptions(kubernetes.WithRolloutTimeout(timeout))
}

/*
WithOwnedCRDCleanup removes Argo CD CRDs on cleanup only if they did not
exist before this stack started. With beforeInstall set, those owned CRDs
are also cleared before install to remove leftovers from previous failed
runs while preserving CRDs that predated this run.
*/
func WithOwnedCRDCleanup(beforeInstall bool) Option {
	return func(config *config) {
		config.cleanupCRDs = true
		config.clearCRDs = beforeInstall
	}
}

/*
NewStack creates an Argo CD quickstart on a Docker-backed k3s cluster.
*/
func NewStack(name string, options ...Option) (*Stack, error) {
	config := &config{namespace: "argocd"}
	for _, option := range options {
		option(config)
	}

	clusterOptions := []kubernetes.Option{
		kubernetes.WithNamespace(config.namespace),
		kubernetes.WithRolloutTimeout(4 * time.Minute),
	}
	if config.kubeconfigPath != "" {
		clusterOptions = append(clusterOptions, kubernetes.WithKubeconfigPath(config.kubeconfigPath))
	}
	clusterOptions = append(clusterOptions, config.clusterOptions...)

	cluster, err := kubernetes.NewCluster(name+"-kubernetes", clusterOptions...)
	if err != nil {
		return nil, err
	}

	return &Stack{
		name:        name,
		namespace:   config.namespace,
		cluster:     cluster,
		manifests:   config.manifests,
		preexisting: map[string]bool{},
		cleanupCRDs: config.cleanupCRDs,
		clearCRDs:   config.clearCRDs,
	}, nil
}

func (s *Stack) Name() string {
	return s.name
}

func (s *Stack) SetLabels(labels map[string]string) {
	s.cluster.SetLabels(labels)
}

func (s *Stack) Create(ctx context.Context) error {
	if err := s.cluster.Create(ctx); err != nil {
		return err
	}

	if err := s.recordCRDs(ctx); err != nil {
		s.cluster.Cleanup(context.WithoutCancel(ctx))
		return err
	}
	if s.cleanupCRDs && s.clearCRDs {
		s.deleteOwnedCRDs(ctx)
	}

	if _, err := s.cluster.Kubectl(ctx, "apply", "--server-side", "-n", s.namespace, "-f", installManifest); err != nil {
		s.cluster.Cleanup(context.WithoutCancel(ctx))
		return fmt.Errorf("failed to install argo cd: %w", err)
	}
	if err := s.waitForCRDs(ctx); err != nil {
		s.cluster.Cleanup(context.WithoutCancel(ctx))
		return err
	}
	if err := s.cluster.WaitForRollouts(ctx); err != nil {
		s.cluster.Cleanup(context.WithoutCancel(ctx))
		return err
	}
	if err := s.applyUserManifests(ctx); err != nil {
		s.cluster.Cleanup(context.WithoutCancel(ctx))
		return err
	}

	return nil
}

func (s *Stack) Cleanup(ctx context.Context) error {
	s.deleteUserManifests(ctx)
	if s.cleanupCRDs {
		s.deleteOwnedCRDs(ctx)
	}
	return s.cluster.Cleanup(ctx)
}

func (s *Stack) Env() map[string]string {
	return s.cluster.Env()
}

func (s *Stack) Endpoints() map[string]string {
	return s.cluster.Endpoints()
}

func (s *Stack) Logs(ctx context.Context) (logs.LogStreams, error) {
	return s.cluster.Logs(ctx)
}

func (s *Stack) Kubectl(ctx context.Context, args ...string) ([]byte, error) {
	return s.cluster.Kubectl(ctx, args...)
}

func (s *Stack) WriteKubeconfig(ctx context.Context, path string) (string, error) {
	return s.cluster.WriteKubeconfig(ctx, path)
}

func (s *Stack) SSHAddress() string {
	return s.cluster.SSHAddress()
}

func (s *Stack) Status(ctx context.Context) ([]byte, error) {
	return s.cluster.Status(ctx)
}

func (s *Stack) RegistryAddress() string {
	return s.cluster.RegistryAddress()
}

func (s *Stack) RegistryInternalAddress() string {
	return s.cluster.RegistryInternalAddress()
}

func (s *Stack) RegistryImage(image string) string {
	return s.cluster.RegistryImage(image)
}

func (s *Stack) RegistryDockerConfigJSON() ([]byte, error) {
	return s.cluster.RegistryDockerConfigJSON()
}

func (s *Stack) RegistryEnv() map[string]string {
	return s.cluster.RegistryEnv()
}

func (s *Stack) PushImage(ctx context.Context, localImage string, clusterImage string) (kubernetes.PushedImage, error) {
	return s.cluster.PushImage(ctx, localImage, clusterImage)
}

func (s *Stack) BuildAndPushImage(ctx context.Context, dockerfile string, clusterImage string) (kubernetes.PushedImage, string, error) {
	return s.cluster.BuildAndPushImage(ctx, dockerfile, clusterImage)
}

func (s *Stack) recordCRDs(ctx context.Context) error {
	for _, name := range crdNames {
		_, err := s.cluster.Kubectl(ctx, "get", "crd", name)
		s.preexisting[name] = err == nil
	}

	return nil
}

func (s *Stack) waitForCRDs(ctx context.Context) error {
	for _, name := range crdNames {
		if _, err := s.cluster.Kubectl(ctx, "wait", "--for", "condition=Established", "crd/"+name, "--timeout", "2m"); err != nil {
			return fmt.Errorf("argo cd crd %s did not become established: %w", name, err)
		}
	}

	return nil
}

func (s *Stack) deleteOwnedCRDs(ctx context.Context) {
	for _, name := range crdNames {
		if s.preexisting[name] {
			continue
		}
		_, _ = s.cluster.Kubectl(ctx, "delete", "crd", name, "--ignore-not-found=true")
	}
}

func (s *Stack) applyUserManifests(ctx context.Context) error {
	for _, manifest := range s.manifests {
		if err := applyManifest(ctx, s.cluster, s.namespace, manifest); err != nil {
			return err
		}
	}

	return nil
}

func (s *Stack) deleteUserManifests(ctx context.Context) {
	for i := len(s.manifests) - 1; i >= 0; i-- {
		_ = deleteManifest(ctx, s.cluster, s.namespace, s.manifests[i])
	}
}

func applyManifest(ctx context.Context, cluster *kubernetes.Cluster, namespace string, manifest string) error {
	if isURL(manifest) {
		_, err := cluster.Kubectl(ctx, "apply", "-n", namespace, "-f", manifest)
		return err
	}

	return walkManifestFiles(manifest, func(path string) error {
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		_, err = cluster.ApplyYAML(ctx, contents)
		return err
	})
}

func deleteManifest(ctx context.Context, cluster *kubernetes.Cluster, namespace string, manifest string) error {
	if isURL(manifest) {
		_, err := cluster.Kubectl(ctx, "delete", "-n", namespace, "-f", manifest, "--ignore-not-found=true")
		return err
	}

	return walkManifestFiles(manifest, func(path string) error {
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		_, err = cluster.DeleteYAML(ctx, contents)
		return err
	})
}

func walkManifestFiles(path string, fn func(string) error) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fn(path)
	}

	return filepath.WalkDir(path, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		return fn(path)
	})
}

func isURL(value string) bool {
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}

var _ scaffold.Service = (*Stack)(nil)
